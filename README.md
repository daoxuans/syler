syler
=====

Syler Hua Wei Portal Middleware

Syler是一个支持华为Portal协议的开源网络管理中间件，支持协议包括

* 华为Portal 1.0 协议
* 华为Portal 2.0 协议

## 已测试设备

### 华为
S3700, AR 1220
### 华三
WX5000

## 下载
### 从源码安装

* go get github.com/daoxuans/syler

## Login接口
	通过调用相关接口上层应用可以实现不同ip、不同用户的上线和下线
	接口地址 http://12.34.56.78/login
	
	接口说明 调用该接口实现某个用户（IP）的上线功能
	
	接口参数
	
	userip,必填,用户IP，待认证的用户的上网IP
	nasip,必填,网络接入设备的IP
	username，可选，用户登录用户名，当随机用户名配置项打开时，该项为可选项；否则为必填
	userpwd，可选，用户登录密码，当随机用户名配置项打开时，该项为可选项；否则为必填
	timeout，可选，用户上网允许时长，当为空时，不限制用户时长
## Logout接口
	接口实现用户的下线功能
	接口地址：http://12.34.56.78/logout
	接口说明：调用该接口实现某个用户（IP）的下线功能
	接口参数：
	userip，必填，用户IP，待下线的用户的上网IP
	nasip，必填，网络接入设备的IP

## 回调接口
	异常下线回调接口由上层应用实现，主要完成用户在交换机内部出现异常情况，交换机将用户下线，并通知portal后的业务逻辑处理。
	异常下线接口在配置文件中配置，Syler通过Get方式调用该接口，并返回一下参数：
	userip：下线用户IP
	nasip：相应NAS设备IP
## 认证测试：
127.0.0.1:8080/login?userip=1.1.1.1&nasip=192.168.0.21&username=123&userpwd=123

## syler.toml
### syler.toml是syler程序的主要配置文件，放置在和syler同级的目录下
	[radius]
	        port = 1812
	        acc_port = 1813
	        secret = "syler"
	        enabled = true   #打开自身集成的radius，用户输入任意账户都可以认证成功
	[http]
	        port = 8080
	[huawei]
	        port = 50100
	        version = 2
	        secret = "syler"
	        nas_port = 2000
	        domain = "1.cn"                     #不为空则加到用户名后缀 @domain
	        timeout = 15
	[basic]
	        callback_logout = "http://10.10.100.68"   #当交换机向服务器发送用户下线等主动消息时，服务器转发的目的url
	        logfile = "./debug.log"             #日志文件
	        remote_ip_as_user_ip = true         #是否使用请求的L3 IP作为用户ip，否则请求需要带上userip参数
	        nas_ip = "10.10.100.253"            #强制所有请求的nasip为该值，否则为请求头中的nasip
