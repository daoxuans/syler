package component

import (
	"fmt"
	"net"

	"daoxuans/syler/huawei/portal"
	v1 "daoxuans/syler/huawei/portal/v1"
	v2 "daoxuans/syler/huawei/portal/v2"
	"daoxuans/syler/logger"

	"github.com/spf13/viper"
)

func StartHuawei() {
	log := logger.GetLogger()

	portal.RegisterFallBack(func(msg portal.Message, src net.IP) {
		log.Println(" type: ", msg.Type())
		if msg.Type() == portal.NTF_LOGOUT {
			NotifyLogout(msg, src)
		}
	})
	if viper.GetInt("huawei.version") == 1 {
		portal.SetVersion(new(v1.Version))
	} else {
		portal.SetVersion(new(v2.Version))
	}

	log.Printf("listen portal server on %d", viper.GetInt("huawei.port"))

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
		log.Printf("got a logout notification from nas %s, but userip is nil", basip)
		return
	}

	log.Printf("got a logout notification of %s from nas %s", userip, basip)
	portal.AckNtfLogout(userip, viper.GetString("huawei.secret"), basip, viper.GetInt("huawei.nas_port"), msg.SerialId(), msg.ReqId())
}
