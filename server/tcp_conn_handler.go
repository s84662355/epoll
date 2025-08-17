package server

import (
	"bufio"
	"bytes"
	"context"
	"net"
	"net/http"

	"go.uber.org/zap" // 高性能日志库

	"proxy_server/log"
)

func (m *manager) handlerTcpConn(ctx context.Context, conn net.Conn) {
	buffer := m.bytePool.Get().([]byte)

	n, err := conn.Read(buffer)
	if err != nil {
		m.bytePool.Put(buffer)
		log.Error("[tcp_conn_handler] 首次读取数据失败", zap.Error(err))
		return
	}

	// socks5
	if n <= 8 {
		m.bytePool.Put(buffer)
		m.socksTcpConn(ctx, conn)
		return
	} else { // http
		bufferReader := bytes.NewBuffer(buffer[:n])
		reader := bufio.NewReader(bufferReader)
		req, err := http.ReadRequest(reader)
		m.bytePool.Put(buffer)
		if err != nil {
			log.Error("[tcp_conn_handler] http代理协议解析失败", zap.Error(err))
			return
		}
		reader = nil
		bufferReader = nil

		m.httpTcpConn(ctx, conn, req)
		return
	}
}
