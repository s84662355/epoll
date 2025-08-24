package server

import (
	"bytes"
	"context"
	//"encoding/base64"
	"fmt"
	"net"
	"net/http"
	//"strings"

	"go.uber.org/zap"

	"proxy_server/log"
)

func (m *manager) httpTcpConn(ctx context.Context, conn net.Conn, req *http.Request) {
	// auth := req.Header.Get("Proxy-Authorization")
	// auth = strings.Replace(auth, "Basic ", "", 1)
	// authData, err := base64.StdEncoding.DecodeString(auth)
	// if err != nil {
	// 	log.Error("[tcp_conn_handler] http代理Proxy-Authorization获取失败", zap.Error(err), zap.Any("auth", auth))
	// 	if _, err = conn.Write([]byte("HTTP/1.1 407 Proxy Authorization Required\r\nProxy-Authenticate: Basic realm=\"Secure Proxys\"\r\n\r\n")); err != nil {
	// 		return
	// 	}
	// 	return
	// }

	// userPasswdPair := strings.Split(string(authData), ":")
	// if len(userPasswdPair) != 2 {
	// 	log.Error("[tcp_conn_handler] http代理账号密码错误", zap.Any("authData", authData))
	// 	if _, err = conn.Write([]byte("HTTP/1.1 407 Proxy Authorization Required\r\nProxy-Authenticate: Basic realm=\"Secure Proxys\"\r\n\r\n")); err != nil {
	// 		return
	// 	}
	// 	return
	// }

	// proxyServerConn := conn.LocalAddr().(*net.TCPAddr)
	// proxyUserName := userPasswdPair[0]
	// proxyPassword := userPasswdPair[1]
	// proxyServerIpStr := proxyServerConn.IP.String()
	// if err := m.Valid(ctx, proxyUserName, proxyPassword, proxyServerIpStr); err != nil {
	// 	log.Error("[tcp_conn_handler] http代理鉴权失败", zap.Error(err))
	// 	if _, err = conn.Write([]byte("HTTP/1.1 407 Proxy Authorization Required\r\nProxy-Authenticate: Basic realm=\"Secure Proxys\"\r\n\r\n")); err != nil {
	// 		return
	// 	}
	// 	return
	// }

	address := req.Host
	_, port, _ := net.SplitHostPort(req.Host)
	if req.Method == "CONNECT" {
		if port == "" {
			address = fmt.Sprint(req.Host, ":", 443)
		}
	} else {
		if port == "" {
			address = fmt.Sprint(req.Host, ":", 80)
		}
	}

	var target net.Conn
	target, err := DialContext(ctx, "tcp", address)
	if err != nil {
		log.Error("[tcp_conn_handler] 创建目标连接失败", zap.Any("target_addr", address))
		if _, err = conn.Write([]byte("HTTP/1.1 503 Service Unavailable\r\n\r\n")); err != nil {
			return
		}
		return
	}
	defer target.Close()

	if req.Method == "CONNECT" {
		if _, err := conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
			return
		}
	} else {
		req.Header.Del("Proxy-Authorization")
		var buf bytes.Buffer
		if err := req.Write(&buf); err != nil {
			log.Error("[tcp_conn_handler] 清除Proxy-Authorization", zap.Error(err))
			return
		}
		if _, err := target.Write(buf.Bytes()); err != nil {
			log.Error("[tcp_conn_handler] 转发http请求数据失败", zap.Error(err))
			return
		}
	}

	if epl, err := MkEpoll(); err != nil {
		log.Error("[tcp_conn_handler] 初始化epoll", zap.Error(err))
		return
	} else {
		defer epl.Close()

		if err := epl.Add(conn, 1); err != nil {
			log.Error("[tcp_conn_handler] 连接1添加到epoll失败", zap.Error(err))
			return
		}

		if err := epl.Add(target, 2); err != nil {
			log.Error("[tcp_conn_handler] 连接2添加到epoll失败", zap.Error(err))
			return
		}
		log.Info("[tcp_conn_handler] 开始传输数据", zap.Any("target_addr", address))

		done := make(chan struct{})
		defer func() {
			for range done {
				/* code */
			}
			log.Info("[tcp_conn_handler]  传输数据结束", zap.Any("target_addr", address))
		}()
		go func() {
			defer close(done)
			buf := make([]byte, 32*1024)
			for {
				if events, err := epl.Wait(); err != nil {
					return
				} else {
					for _, event := range events {
						if event.Pad == 1 {
							if n, err := conn.Read(buf); err != nil {
								return
							} else {
								if _, err := target.Write(buf[:n]); err != nil {
									return
								}
							}
						} else {
							if n, err := target.Read(buf); err != nil {
								return
							} else {
								if _, err := conn.Write(buf[:n]); err != nil {
									return
								}
							}
						}
					}
				}
			}
		}()

		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				epl.Close()
				<-done
				return

			}
		}

	}
}
