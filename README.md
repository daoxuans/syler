syler
=====

Syler Hua Wei Portal Middleware

Syler是一个支持华为Portal协议的开源网络管理中间件，支持协议包括

* 华为Portal 1.0 协议
* 华为Portal 2.0 协议
* 短信验证码认证

## 已测试设备

### 华为
S3700, AR 1220
### 华三
WX5000

## 下载
### 从源码安装

* go get github.com/syler

## Login接口
    通过调用相关接口上层应用可以实现不同ip、不同用户的上线和下线
    接口地址 http://12.34.56.78/login

    接口说明 调用该接口实现某个用户（IP）的上线功能

    接口参数

    userip,必填,用户IP，待认证的用户的上网IP
    nasip,必填,网络接入设备的IP
    username，必填，用户手机号（启用短信验证码时）或登录用户名
    userpwd，必填，短信验证码（启用短信验证码时）或登录密码

## 短信验证码接口
    接口地址：http://12.34.56.78/api/sendcode
    接口说明：发送短信验证码
    请求方式：POST
    请求参数：
    phone，必填，用户手机号（11位）

## Logout接口
    接口实地址：http://12.34.56.78/logout
    接口说明：调用该接口实现某个用户（IP）的下线功能
    接口参数：
    userip，必填，用户IP，待下线的用户的上网IP
    nasip，必填，网络接入设备的IP

## 认证测试
账号密码认证：
```
127.0.0.1:8080/login?userip=1.1.1.1&nasip=192.168.0.21&username=123&userpwd=123
```

短信验证码认证：
```
# 1. 发送验证码
curl -X POST http://127.0.0.1:8080/api/sendcode \
     -H "Content-Type: application/json" \
     -d '{"phone":"13800138000"}'

# 2. 使用验证码登录
127.0.0.1:8080/login?userip=1.1.1.1&nasip=192.168.0.21&username=13800138000&userpwd=123456
```

## syler.toml 配置说明
### syler.toml是syler程序的主要配置文件，放置在和syler同级的目录下

```toml
[http]
port=8080                    # HTTP服务端口
remote_ip_as_user_ip=false   # 是否使用请求的L3 IP作为用户ip
nas_ip=""                    # 强制所有请求的nasip为该值
white_list=""               # IP白名单，多个IP用逗号分隔

[portal]
port=50100                  # Portal服务端口
version=2                   # Portal协议版本
secret="syler"             # 共享密钥
nas_port=2000              # NAS端口
domain=""                  # 用户名后缀域名

[sms]
provider=""                # 短信服务商：aliyun/tencent
access_key=""             # 访问密钥ID
secret_key=""             # 访问密钥密码
sign_name=""              # 短信签名
template_code=""          # 短信模板ID
region="ap-guangzhou"     # 地区（腾讯云）
sdk_app_id=""            # SDK应用ID（腾讯云）

[redis]
addr="localhost:6379"     # Redis服务器地址
password=""              # Redis密码

[basic]
logfile="./syler.log"    # 日志文件路径
```

## 注意事项
1. 短信验证码功能需要配置 SMS 服务商信息
2. 验证码存储需要配置 Redis 服务
3. 如需限制访问IP，可配置 http.white_list
