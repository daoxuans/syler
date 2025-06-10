package server

import (
	"fmt"
	"net"

	"github.com/sirupsen/logrus"

	"syler/internal/logger"
	"syler/internal/portal"
	v1 "syler/internal/portal/v1"
	v2 "syler/internal/portal/v2"

	"github.com/spf13/viper"
)

type PortalConfig struct {
	Secret  string
	NasPort int
	Version int
	Port    int
	Host    string
}

func LoadPortalConfig() PortalConfig {
	return PortalConfig{
		Secret:  viper.GetString("portal.secret"),
		NasPort: viper.GetInt("portal.nas_port"),
		Version: viper.GetInt("portal.version"),
		Port:    viper.GetInt("portal.port"),
		Host:    viper.GetString("portal.host"),
	}
}

var portalConfig PortalConfig

func StartPortal() {
	log := logger.GetLogger()

	portalConfig = LoadPortalConfig()

	portal.RegisterFallBack(func(msg portal.Message, src net.IP) {
		if msg.Type() == portal.NTF_LOGOUT {
			log.WithFields(logrus.Fields{
				"message_type": "NTF_LOGOUT",
				"source_ip":    src.String(),
			}).Debug("Received portal logout notification")
			NotifyLogout(msg, src)
		}
	})
	if portalConfig.Version == 1 {
		portal.SetVersion(new(v1.Version))
	} else {
		portal.SetVersion(new(v2.Version))
	}

	log.WithFields(logrus.Fields{
		"host": portalConfig.Host,
		"port": portalConfig.Port,
	}).Info("Starting portal server")

	addr := fmt.Sprintf("%s:%d", portalConfig.Host, portalConfig.Port)
	portal.ListenAndService(addr)
}

func Challenge(userip net.IP, basip net.IP) (response portal.Message, err error) {
	return portal.Challenge(userip, portalConfig.Secret, basip, portalConfig.NasPort)
}

func Auth(userip net.IP, basip net.IP, username, userpwd []byte) (err error) {
	var res portal.Message
	if res, err = Challenge(userip, basip); err == nil {
		if cres, ok := res.(portal.ChallengeRes); ok {
			res, err = portal.ChapAuth(userip, portalConfig.Secret, basip, portalConfig.NasPort, username, userpwd, res.ReqId(), cres.GetChallenge())
			if err == nil {
				_, err = portal.AffAckAuth(userip, portalConfig.Secret, basip, portalConfig.NasPort, res.SerialId(), res.ReqId())
			}
		}
	}
	return
}

func Logout(userip net.IP, basip net.IP) (response portal.Message, err error) {
	return portal.Logout(userip, portalConfig.Secret, basip, portalConfig.NasPort)
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
	portal.AckNtfLogout(userip, portalConfig.Secret, basip, portalConfig.NasPort, msg.SerialId(), msg.ReqId())
}
