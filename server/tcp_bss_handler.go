package server

import (
	"context"
	//"encoding/base64"
	"encoding/binary"
	"fmt"
	"net"
	//"strings"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"

	"go.uber.org/zap"

	"proxy_server/log"
)

// Base64Encode 对二进制数据进行Base64编码
func Base64Encode(data []byte) string {
	// 使用标准Base64编码（兼容RFC 4648）
	return base64.StdEncoding.EncodeToString(data)
}

// Base64Decode 对Base64编码的字符串进行解码
func Base64Decode(encoded string) ([]byte, error) {
	// 解码Base64字符串，返回原始二进制数据
	return base64.StdEncoding.DecodeString(encoded)
}

// 加密数据：使用AES-GCM算法
func encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, err
	}

	// GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 生成随机非ce（12字节推荐值）
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// 加密并添加认证标签
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// 发送加密数据：先发送长度（4字节大端序），再发送加密内容
func sendEncrypted(conn net.Conn, data []byte) error {
	// 加密数据
	encrypted, err := encrypt(data)
	if err != nil {
		return err
	}

	// 先发送长度（4字节）
	lengthBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBuf, uint32(len(encrypted)))
	if _, err := conn.Write(lengthBuf); err != nil {
		return err
	}

	// 再发送加密内容
	_, err = conn.Write(encrypted)
	return err
}

// 解密数据：对应AES-GCM加密
func decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 分离非ce和密文
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("密文太短")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// 解密
	return gcm.Open(nil, nonce, ciphertext, nil)
}

var encryptionKey = []byte("0123456789abcdef") // 16字节，AES-128
func (m *manager) tcpBssHandler(ctx context.Context, conn net.Conn) {
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

	if addressBYte, err := Base64Decode(address); err != nil {
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, uint32(len(fmt.Sprintf("base64解码失败"))))
		buf = append(buf, []byte(fmt.Sprintf("base64解码失败"))...)
		conn.Write(buf)
		log.Error("[tcp_oks_handler] base64解码失败", zap.Error(err))
		return
	} else {
		address = string(addressBYte)
	}

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

							// 读取长度（4字节）
							lengthBuf := make([]byte, 4)
							if _, err := io.ReadFull(conn, lengthBuf); err != nil {
								log.Error("[tcp_bss_handler]	读取加密数据长度错误", zap.Error(err))
								return
							}
							dataLen := binary.BigEndian.Uint32(lengthBuf)
							if dataLen > 32*1024 {
								log.Error("[tcp_bss_handler]	读取加密数据长度大于32*1024 ", zap.Error(err))
								return
							}

							// 读取加密内容
							encrypted := make([]byte, dataLen)
							if _, err := io.ReadFull(conn, encrypted); err != nil {
								log.Error("[tcp_bss_handler]	读取加密数据错误 ", zap.Error(err))
								return
							} else {
								if dbuf, err := decrypt(encrypted); err != nil {
									log.Error("[tcp_bss_handler] 解析加密数据错误 ", zap.Error(err))
									return
								} else {
									if _, err := target.Write(dbuf); err != nil {
										log.Error("[tcp_bss_handler] 发送数据错误 ", zap.Error(err))
										return
									}
								}
							}
							// if n, err := conn.Read(buf); err != nil {
							// 	return
							// } else {
							// 	if _, err := target.Write(buf[:n]); err != nil {
							// 		return
							// 	}
							// }

						} else {
							if n, err := target.Read(buf); err != nil {
								return
							} else {
								if err := sendEncrypted(conn, buf[:n]); err != nil {
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
