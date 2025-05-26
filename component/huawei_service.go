package component

import (
	"fmt"
	"log"
	"net"

	"daoxuans/syler/config"
	"daoxuans/syler/huawei/portal"
	v1 "daoxuans/syler/huawei/portal/v1"
	v2 "daoxuans/syler/huawei/portal/v2"
)

func StartHuawei() {
	portal.RegisterFallBack(func(msg portal.Message, src net.IP) {
		log.Println(" type: ", msg.Type())
		if msg.Type() == portal.NTF_LOGOUT {
			NotifyLogout(msg, src)
		}
	})
	if *config.HuaweiVersion == 1 {
		portal.SetVersion(new(v1.Version))
	} else {
		portal.SetVersion(new(v2.Version))
	}

	log.Printf("listen portal on %d\n", *config.HuaweiPort)

	portal.ListenAndService(fmt.Sprintf(":%d", *config.HuaweiPort))
}

func Challenge(userip net.IP, basip net.IP) (response portal.Message, err error) {
	return portal.Challenge(userip, *config.HuaweiSecret, basip, *config.HuaweiNasPort)
}

func Auth(userip net.IP, basip net.IP, username, userpwd []byte) (err error) {
	var res portal.Message
	if res, err = Challenge(userip, basip); err == nil {
		if cres, ok := res.(portal.ChallengeRes); ok {
			res, err = portal.ChapAuth(userip, *config.HuaweiSecret, basip, *config.HuaweiNasPort, username, userpwd, res.ReqId(), cres.GetChallenge())
			if err == nil {
				_, err = portal.AffAckAuth(userip, *config.HuaweiSecret, basip, *config.HuaweiNasPort, res.SerialId(), res.ReqId())
			}
		}
	}
	return
}

func Logout(userip net.IP, basip net.IP) (response portal.Message, err error) {
	return portal.Logout(userip, *config.HuaweiSecret, basip, *config.HuaweiNasPort)
}

func NotifyLogout(msg portal.Message, basip net.IP) {
	userip := msg.UserIp()
	if userip == nil {
		log.Printf("got a logout notification from nas %s, but userip is nil\n", basip)
		return
	}

	log.Printf("got a logout notification of %s from nas %s\n", userip, basip)
	portal.AckNtfLogout(userip, *config.HuaweiSecret, basip, *config.HuaweiNasPort, msg.SerialId(), msg.ReqId())
}
