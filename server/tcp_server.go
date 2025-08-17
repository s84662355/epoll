package server

import (
	"context"
	"net"
	"sync"
	"time"

	"go.uber.org/zap" // 高性能日志库

	"proxy_server/config"
	"proxy_server/log"
)
///events_mask = unix.EPOLLIN | unix.EPOLLOUT | unix.EPOLLHUP | unix.EPOLLERR

func (m *manager) initTcpListener() {
	conf := config.GetConf()

	m.tcpListener = map[string]net.Listener{}
	for _, v := range conf.TcpListenerAddress {
		listener, err := net.Listen("tcp", v)
		if err != nil {
			log.Panic("[tcp_server] 初始化tcp监听服务失败", zap.Error(err))
		}
		m.tcpListener[v] = listener
	}
}

func (m *manager) tcpAccept(ctx context.Context) {
	wg := &sync.WaitGroup{}
	defer func() {
		wg.Wait()
		timeOutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		<-timeOutCtx.Done()
	}()

	for _, tcpListener := range m.tcpListener {
		wg.Add(1)
		go func(tcpListener net.Listener) {
			defer wg.Done()
			m.tcpListenerAccept(ctx, tcpListener)
		}(tcpListener)
	}
}

func (m *manager) tcpListenerAccept(ctx context.Context, tcpListener net.Listener) {
	addr := tcpListener.Addr()
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	log.Info("[tcp_server]  开启tcp监听服务", zap.Any("addr", addr.String()))
	for m.isRun.Load() {
		// 接受客户端的连接
		conn, err := tcpListener.Accept()
		if err != nil {
			// 若接受连接出错，记录错误信息并继续等待下一个连接
			log.Error("[tcp_server]  tcp监听服务Accept", zap.Error(err), zap.Any("addr", addr.String()))
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			done := make(chan struct{})
			defer func() {
				conn.Close()
				for range done {
				}
			}()

			go func() {
				defer close(done)
				m.handlerTcpConn(ctx, conn)
			}()

			select {
			case <-ctx.Done():
			case <-done:
			}
		}()

	}
}
