package server

import (
	"context"
	//"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	//"strings"
	"go.uber.org/zap"

	"proxy_server/log"
)

func (m *manager) tcpOksHandler(ctx context.Context, conn net.Conn) {
	buf := make([]byte, 4)
	// 使用io.ReadFull确保读取满4个字节

	n, err := io.ReadFull(conn, buf)
	if err != nil {
		log.Error("[tcp_oks_handler] 读取地址长度失败", zap.Error(err))
		return
	}

	dataLen := binary.BigEndian.Uint32(buf)
	if dataLen > 10000 {
		log.Error("[tcp_oks_handler] 读取地址长度过大", zap.Any("dataLen", dataLen))
		return
	}

	buf = make([]byte, dataLen)
	n, err = io.ReadFull(conn, buf)
	if err != nil {
		log.Error("[tcp_oks_handler] 读取地址失败", zap.Error(err))
		return
	}

	address := string(buf[:n])

	var target net.Conn
	target, err = DialContext(ctx, "tcp", address)
	if err != nil {
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, uint32(len(fmt.Sprintf("连接地址%s失败", address))))
		buf = append(buf, []byte(fmt.Sprintf("连接地址%s失败", address))...)
		conn.Write(buf)
		log.Error("[tcp_oks_handler] 创建目标连接失败", zap.Any("target_addr", address), zap.Error(err))
		if _, err = conn.Write(buf); err != nil {
			return
		}
		return
	}
	defer target.Close()

	buf = make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(len(fmt.Sprintf("ok"))))
	buf = append(buf, []byte(fmt.Sprintf("ok"))...)
	if _, err = conn.Write(buf); err != nil {
		return
	}

	if epl, err := MkEpoll(); err != nil {
		log.Error("[tcp_oks_handler] 初始化epoll", zap.Error(err))
		return
	} else {
		defer epl.Close()

		if err := epl.Add(conn, 1); err != nil {
			log.Error("[tcp_oks_handler] 连接1添加到epoll失败", zap.Error(err))
			return
		}

		if err := epl.Add(target, 2); err != nil {
			log.Error("[tcp_oks_handler] 连接2添加到epoll失败", zap.Error(err))
			return
		}
		log.Info("[tcp_oks_handler] 开始传输数据", zap.Any("target_addr", address))

		done := make(chan struct{})
		defer func() {
			for range done {
				/* code */
			}
			log.Info("[tcp_oks_handler]  传输数据结束", zap.Any("target_addr", address))
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
