package v1

import (
	"fmt"
	"net"
	"testing"
)

func TestChallange(t *testing.T) {
	v := &Version{}
	ip := net.IPv4(192, 168, 56, 2)
	fmt.Println(v.NewChallenge(ip, "it is a secret"))
}

func TestAuth(t *testing.T) {
	v := &Version{}
	ip := net.IPv4(192, 168, 56, 2)
	// TODO: 需要challenge的值才能正确生成Auth消息
	challenge := []byte("challenge")
	fmt.Println(v.NewAuth(ip, "it is a secret", []byte("刘铭"), []byte("456"), uint16(1234), challenge))
}

func TestRunChallange(t *testing.T) {
	// TODO: 实现Challenge函数或修改测试
	Challenge(net.IPv4(192, 168, 1, 1), "it is a secret", net.IPv4(192, 168, 56, 2))
}

func TestUnmarshal(t *testing.T) {
	v := &Version{}
	msg := v.Unmarshall([]byte{0x01, 0x02, 0x00, 0x00, 0x6f, 0x3c, 0x00, 0x06, 0xc0, 0xa8, 0x0a, 0xfe, 0x00, 0x00, 0x00, 0x01, 0x03, 0x12, 0xef, 0x47, 0x25, 0x3d, 0xc5, 0x19, 0x41, 0xb7, 0x63, 0x97, 0x35, 0x07, 0x75, 0xe7, 0x3d, 0x95})
	fmt.Println(msg)
}
