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

func StartHuawei() {
	log := logger.GetLogger()

	portal.RegisterFallBack(func(msg portal.Message, src net.IP) {
		if msg.Type() == portal.NTF_LOGOUT {
			log.WithFields(logrus.Fields{
				"message_type": "NTF_LOGOUT",
				"source_ip":    src.String(),
			}).Debug("Received portal logout notification")
			NotifyLogout(msg, src)
		}
	})
	if viper.GetInt("huawei.version") == 1 {
		portal.SetVersion(new(v1.Version))
	} else {
		portal.SetVersion(new(v2.Version))
	}

	log.WithFields(logrus.Fields{
		"port": viper.GetInt("huawei.port"),
	}).Info("Starting portal server")

	portal.ListenAndService(fmt.Sprintf(":%d", viper.GetInt("huawei.port")))
}

func Challenge(userip net.IP, basip net.IP) (response portal.Message, err error) {
	return portal.Challenge(userip, viper.GetString("huawei.secret"), basip, viper.GetInt("huawei.nas_port"))
}

func Auth(userip net.IP, basip net.IP, username, userpwd []byte) (err error) {
	var res portal.Message
	if res, err = Challenge(userip, basip); err == nil {
		if cres, ok := res.(portal.ChallengeRes); ok {
			res, err = portal.ChapAuth(userip, viper.GetString("huawei.secret"), basip, viper.GetInt("huawei.nas_port"), username, userpwd, res.ReqId(), cres.GetChallenge())
			if err == nil {
				_, err = portal.AffAckAuth(userip, viper.GetString("huawei.secret"), basip, viper.GetInt("huawei.nas_port"), res.SerialId(), res.ReqId())
			}
		}
	}
	return
}

func Logout(userip net.IP, basip net.IP) (response portal.Message, err error) {
	return portal.Logout(userip, viper.GetString("huawei.secret"), basip, viper.GetInt("huawei.nas_port"))
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
	portal.AckNtfLogout(userip, viper.GetString("huawei.secret"), basip, viper.GetInt("huawei.nas_port"), msg.SerialId(), msg.ReqId())
}
