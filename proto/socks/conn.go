package socks

import (
	"fmt"
	"net"

	"golang.org/x/net/proxy"
)

// GetConn 通过SOCKS5代理建立到目标地址的连接
// s: SOCKS5代理服务器URL，格式如 "user:password@host:port"
// targetAddr: 要通过代理连接的目标地址，格式如 "host:port"
// 返回值: 已建立的网络连接和可能的错误
func GetConn(s string, targetAddr string) (net.Conn, error) {
	// 1. 解析代理URL
	auth, server, err := ParseURL(s) // 解析出认证信息和服务器地址
	if err != nil {
		return nil, fmt.Errorf("解析SOCKS5代理URL失败: %w", err)
	}

	// 2. 创建SOCKS5拨号器
	dialer, err := proxy.SOCKS5(
		"tcp",        // 网络类型(tcp)
		server,       // 代理服务器地址(host:port)
		auth,         // 认证信息(nil或Auth)
		proxy.Direct, // 底层拨号器(直接连接)
	)
	if err != nil {
		return nil, fmt.Errorf("创建SOCKS5拨号器失败: %w", err)
	}

	// 3. 通过代理建立到目标地址的连接
	conn, err := dialer.Dial("tcp", targetAddr)
	if err != nil {
		return nil, fmt.Errorf("通过SOCKS5代理连接目标失败: %w", err)
	}

	return conn, nil
}
