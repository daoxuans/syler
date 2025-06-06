package component

import (
	"fmt"
	"net"

	"github.com/sirupsen/logrus"

	"daoxuans/syler/huawei/portal"
	v1 "daoxuans/syler/huawei/portal/v1"
	v2 "daoxuans/syler/huawei/portal/v2"
	"daoxuans/syler/logger"

	"github.com/spf13/viper"
)

type HuaweiConfig struct {
	Secret  string
	NasPort int
	Version int
	Port    int
	Host    string
}

func LoadHuaweiConfig() HuaweiConfig {
	return HuaweiConfig{
		Secret:  viper.GetString("huawei.secret"),
		NasPort: viper.GetInt("huawei.nas_port"),
		Version: viper.GetInt("huawei.version"),
		Port:    viper.GetInt("huawei.port"),
		Host:    viper.GetString("huawei.host"),
	}
}

var huaweiConfig HuaweiConfig

func StartHuawei() {
	log := logger.GetLogger()

	huaweiConfig = LoadHuaweiConfig()

	portal.RegisterFallBack(func(msg portal.Message, src net.IP) {
		if msg.Type() == portal.NTF_LOGOUT {
			log.WithFields(logrus.Fields{
				"message_type": "NTF_LOGOUT",
				"source_ip":    src.String(),
			}).Debug("Received portal logout notification")
			NotifyLogout(msg, src)
		}
	})
	if huaweiConfig.Version == 1 {
		portal.SetVersion(new(v1.Version))
	} else {
		portal.SetVersion(new(v2.Version))
	}

	log.WithFields(logrus.Fields{
		"host": huaweiConfig.Host,
		"port": huaweiConfig.Port,
	}).Info("Starting portal server")

	addr := fmt.Sprintf("%s:%d", huaweiConfig.Host, huaweiConfig.Port)
	portal.ListenAndService(addr)
}

func Challenge(userip net.IP, basip net.IP) (response portal.Message, err error) {
	return portal.Challenge(userip, huaweiConfig.Secret, basip, huaweiConfig.NasPort)
}

func Auth(userip net.IP, basip net.IP, username, userpwd []byte) (err error) {
	var res portal.Message
	if res, err = Challenge(userip, basip); err == nil {
		if cres, ok := res.(portal.ChallengeRes); ok {
			res, err = portal.ChapAuth(userip, huaweiConfig.Secret, basip, huaweiConfig.NasPort, username, userpwd, res.ReqId(), cres.GetChallenge())
			if err == nil {
				_, err = portal.AffAckAuth(userip, huaweiConfig.Secret, basip, huaweiConfig.NasPort, res.SerialId(), res.ReqId())
			}
		}
	}
	return
}

func Logout(userip net.IP, basip net.IP) (response portal.Message, err error) {
	return portal.Logout(userip, huaweiConfig.Secret, basip, huaweiConfig.NasPort)
}

func NotifyLogout(msg portal.Message, basip net.IP) {
	log := logger.GetLogger()

	userip := msg.UserIp()
	if userip == nil {
		log.WithFields(logrus.Fields{
			"nas_ip": basip.String(),
		}).Warn("Received logout notification with nil user IP")
		return
	}

	log.WithFields(logrus.Fields{
		"user_ip": userip.String(),
		"nas_ip":  basip.String(),
	}).Info("Received logout notification")
	portal.AckNtfLogout(userip, huaweiConfig.Secret, basip, huaweiConfig.NasPort, msg.SerialId(), msg.ReqId())
}
