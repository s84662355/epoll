package socks5

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
)

const (
	socks5Version = uint8(5)
)

const (
	NoAuth          = uint8(0)
	NoAcceptable    = uint8(255)
	UserPassAuth    = uint8(2)
	UserAuthVersion = uint8(1)
	AuthSuccess     = uint8(0)
	AuthFailure     = uint8(1)
)

var (
	UserAuthFailed  = fmt.Errorf("用户身份验证失败")
	NoSupportedAuth = fmt.Errorf("没有支持的身份验证机制")
)

const (
	ConnectCommand   = uint8(1)
	BindCommand      = uint8(2)
	AssociateCommand = uint8(3)
	ipv4Address      = uint8(1)
	fqdnAddress      = uint8(3)
	ipv6Address      = uint8(4)
)

const (
	SuccessReply uint8 = iota
	ServerFailure
	RuleFailure
	NetworkUnreachable
	HostUnreachable
	ConnectionRefused
	TtlExpired
	CommandNotSupported
	AddrTypeNotSupported
)

var UnrecognizedAddrType = fmt.Errorf("无法识别的地址类型")

type AddrSpec struct {
	FQDN string
	IP   net.IP
	Port int
}

func (a *AddrSpec) String() string {
	if a.FQDN != "" {
		return fmt.Sprintf("%s (%s):%d", a.FQDN, a.IP, a.Port)
	}
	return fmt.Sprintf("%s:%d", a.IP, a.Port)
}

func (a AddrSpec) Address() string {
	if 0 != len(a.IP) {
		return net.JoinHostPort(a.IP.String(), strconv.Itoa(a.Port))
	}
	return net.JoinHostPort(a.FQDN, strconv.Itoa(a.Port))
}

func ReadDestAddr(bufConn net.Conn) (*AddrSpec, error) {
	bufConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	// 读取版本字节
	header := []byte{0, 0, 0}
	if _, err := io.ReadAtLeast(bufConn, header, 3); err != nil {
		return nil, fmt.Errorf("无法获取命令版本%v", err)
	}

	// 判断版本号
	if header[0] != socks5Version {
		return nil, fmt.Errorf("不支持的命令版本： %v", header[0])
	}

	// 读取目标地址
	dest, err := ReadAddrSpec(bufConn)
	if err != nil {
		return nil, err
	}

	return dest, nil
}

func ReadVersion(conn net.Conn) (uint8, error) {
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	// 读取版本
	version := []byte{0}
	if _, err := conn.Read(version); err != nil {
		return 0, err
	}

	if version[0] != socks5Version {
		err := fmt.Errorf("不支持的SOCKS版本: %v", version)
		return version[0], err
	}
	return version[0], nil
}

// 用于读取方法的数量,并继续验证方法
func ReadMethods(r net.Conn) ([]byte, error) {
	header := []byte{0}
	r.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := r.Read(header); err != nil {
		return nil, err
	}

	r.SetReadDeadline(time.Now().Add(2 * time.Second))
	numMethods := int(header[0])
	methods := make([]byte, numMethods)
	_, err := io.ReadAtLeast(r, methods, numMethods)
	return methods, err
}

// SendReply用于发送回复消息
func SendReply(r net.Conn, resp uint8, addr *AddrSpec) error {
	// 设置地址格式
	var addrType uint8
	var addrBody []byte
	var addrPort uint16
	switch {
	case addr == nil:
		addrType = ipv4Address
		addrBody = []byte{0, 0, 0, 0}
		addrPort = 0

	case addr.FQDN != "":
		addrType = fqdnAddress
		addrBody = append([]byte{byte(len(addr.FQDN))}, addr.FQDN...)
		addrPort = uint16(addr.Port)

	case addr.IP.To4() != nil:
		addrType = ipv4Address
		addrBody = []byte(addr.IP.To4())
		addrPort = uint16(addr.Port)

	case addr.IP.To16() != nil:
		addrType = ipv6Address
		addrBody = []byte(addr.IP.To16())
		addrPort = uint16(addr.Port)

	default:
		return fmt.Errorf("无法格式化地址： %v", addr)
	}

	msg := make([]byte, 6+len(addrBody))
	msg[0] = socks5Version
	msg[1] = resp
	msg[2] = 0 // Reserved
	msg[3] = addrType
	copy(msg[4:], addrBody)
	msg[4+len(addrBody)] = byte(addrPort >> 8)
	msg[4+len(addrBody)+1] = byte(addrPort & 0xff)
	r.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, err := r.Write(msg)
	return err
}

// 读取目标的地址
// 需要地址类型字节，后跟地址和端口
func ReadAddrSpec(r net.Conn) (*AddrSpec, error) {
	d := &AddrSpec{}

	// 获取地址类型
	addrType := []byte{0}
	r.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := r.Read(addrType); err != nil {
		return nil, err
	}

	// 按类型处理
	r.SetReadDeadline(time.Now().Add(2 * time.Second))
	switch addrType[0] {
	case ipv4Address:
		addr := make([]byte, 4)
		if _, err := io.ReadAtLeast(r, addr, len(addr)); err != nil {
			return nil, err
		}
		d.IP = net.IP(addr)

	case ipv6Address:
		addr := make([]byte, 16)
		if _, err := io.ReadAtLeast(r, addr, len(addr)); err != nil {
			return nil, err
		}
		d.IP = net.IP(addr)

	case fqdnAddress:
		if _, err := r.Read(addrType); err != nil {
			return nil, err
		}
		addrLen := int(addrType[0])
		fqdn := make([]byte, addrLen)
		if _, err := io.ReadAtLeast(r, fqdn, addrLen); err != nil {
			return nil, err
		}
		d.FQDN = string(fqdn)

	default:
		return nil, UnrecognizedAddrType
	}

	// 读取端口
	r.SetReadDeadline(time.Now().Add(2 * time.Second))
	port := []byte{0, 0}
	if _, err := io.ReadAtLeast(r, port, 2); err != nil {
		return nil, err
	}
	d.Port = (int(port[0]) << 8) | int(port[1])

	return d, nil
}

func GetUserPassword(conn net.Conn) (string, string, error) {
	// 告诉客户端使用用户/传递身份验证
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Write([]byte{socks5Version, UserPassAuth}); err != nil {
		return "", "", err
	}

	// 获取版本和用户名长度
	header := []byte{0, 0}
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := io.ReadAtLeast(conn, header, 2); err != nil {
		return "", "", err
	}

	if header[0] != UserAuthVersion {
		return "", "", fmt.Errorf("不支持的身份验证版本: %v", header[0])
	}

	// 获取用户名
	userLen := int(header[1])
	user := make([]byte, userLen)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := io.ReadAtLeast(conn, user, userLen); err != nil {
		return "", "", err
	}

	// 获取密码长度
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Read(header[:1]); err != nil {
		return "", "", err
	}

	// 获取密码
	passLen := int(header[0])
	pass := make([]byte, passLen)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := io.ReadAtLeast(conn, pass, passLen); err != nil {
		return "", "", err
	}

	return string(user), string(pass), nil
}
