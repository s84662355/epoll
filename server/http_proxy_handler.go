package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"proxy_server/log"
	"proxy_server/server/sniffing"
	"proxy_server/server/sniffing/tls"
)

func (m *manager) httpTcpConn(ctx context.Context, conn net.Conn, req *http.Request) {
	auth := req.Header.Get("Proxy-Authorization")
	auth = strings.Replace(auth, "Basic ", "", 1)
	authData, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		log.Error("[tcp_conn_handler] http代理Proxy-Authorization获取失败", zap.Error(err), zap.Any("auth", auth))
		if _, err = conn.Write([]byte("HTTP/1.1 407 Proxy Authorization Required\r\nProxy-Authenticate: Basic realm=\"Secure Proxys\"\r\n\r\n")); err != nil {
			return
		}
		return
	}

	userPasswdPair := strings.Split(string(authData), ":")
	if len(userPasswdPair) != 2 {
		log.Error("[tcp_conn_handler] http代理账号密码错误", zap.Any("authData", authData))
		if _, err = conn.Write([]byte("HTTP/1.1 407 Proxy Authorization Required\r\nProxy-Authenticate: Basic realm=\"Secure Proxys\"\r\n\r\n")); err != nil {
			return
		}
		return
	}

	proxyServerConn := conn.LocalAddr().(*net.TCPAddr)
	proxyUserName := userPasswdPair[0]
	proxyPassword := userPasswdPair[1]
	proxyServerIpStr := proxyServerConn.IP.String()
	if _, err := m.Valid(ctx, proxyUserName, proxyPassword, proxyServerIpStr); err != nil {
		log.Error("[tcp_conn_handler] http代理鉴权失败", zap.Error(err))
		if _, err = conn.Write([]byte("HTTP/1.1 407 Proxy Authorization Required\r\nProxy-Authenticate: Basic realm=\"Secure Proxys\"\r\n\r\n")); err != nil {
			return
		}
		return
	}
 

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

	domain := regexpDomain(address)
	if domain != "" { 
	} 
	var domainPointer atomic.Pointer[string]
	domainPointer.Store(&domain)

	var target net.Conn
	target, err = DialContext(ctx, "tcp", address )
	if err != nil {
		log.Error("[tcp_conn_handler] 创建目标连接失败", zap.Any("local_ip", proxyServerIpStr), zap.Any("target_addr", address), zap.Any("user", proxyUserName))
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
 

	var netConn, netTarget io.ReadWriteCloser

	netConn = newConn(conn, CONN_WRITE_TIME, CONN_READ_TIME)
	netTarget = newConn(target, CONN_WRITE_TIME, CONN_READ_TIME)

	byteChan := make(chan []byte, 1)
	defer close(byteChan)

	errCh := make(chan error, 2)
	defer close(errCh)

 
	wg := sync.WaitGroup{}
	defer func() {
	 
		netConn.Close()
		netTarget.Close()
		wg.Wait()

		domain := domainPointer.Load()
		if domain != nil && *domain != "" {
			m.ReportAccessLogToInfluxDB(proxyUserName, *domain, proxyServerConn.String())
		} else {
			hostArr := strings.Split(address, ":")
			if cap(hostArr) > 0 {
				m.ReportAccessLogToInfluxDB(proxyUserName, hostArr[0], proxyServerConn.String())
			} else {
				m.ReportAccessLogToInfluxDB(proxyUserName, address, proxyServerConn.String())
			}
		}

		 
	}()

 

	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := io.CopyBuffer(netTarget, NewLimitedReader(connCtx.ctx, netConn, action), make([]byte, 2*1024))
		errCh <- err
	}()

	go func() {
		defer wg.Done()
		_, err := io.CopyBuffer(netConn, NewLimitedReader(connCtx.ctx, netTarget, action), make([]byte, 2*1024))
		errCh <- err
	}()

	loopTime := 30 * time.Second
	ticker := time.NewTicker(loopTime)
	defer ticker.Stop()

	for {
		ticker.Reset(loopTime)
		select {
 
		case <-ticker.C:
	 
		case err, _ := <-errCh:
			if err != nil {
				log.Error("[tcp_conn_handler] conn close!",
					zap.Error(err),
					zap.Any("username", proxyUserName),
					zap.Any("clientAddr", proxyServerIpStr),
					zap.Any("target_host", address),
				)
			}

			return
		case <-ctx.Done():
			return
		case <-connCtx.ctx.Done():
			return

		}
	}
}
