package server

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/orcaman/concurrent-map/v2"
	"github.com/spf13/viper"

	"google.golang.org/grpc"

	"proxy_server/protobuf"
	"proxy_server/utils/Queue"
	"proxy_server/utils/rabbitMQ"
	"proxy_server/utils/taskConsumerManager"
)

const AcceptAmount = 8

// 使用 sync.OnceValue 确保 manager 只被初始化一次（线程安全）
var newManager = sync.OnceValue(func() *manager {
	m := &manager{
		tcm:            taskConsumerManager.New(), // 任务消费者管理器
		ipConnCountMap: cmap.New[*IpConnCountMapData](),
		userCtxMap:     cmap.New[*connContext](),
	}
	m.isRun.Store(true)
	m.bytePool = sync.Pool{
		// 当池为空时，使用 New 函数创建新对象
		New: func() interface{} {
			return make([]byte, 2*1024)
		},
	}

	return m
})

// manager 结构体管理整个代理服务的核心组件
type manager struct {
	protobuf.UnimplementedAuthServer
	tcm                            *taskConsumerManager.Manager // 任务调度管理器
	tcpListener                    map[string]net.Listener
	isRun                          atomic.Bool
	bytePool                       sync.Pool
	epl *epoll
}

// Start 启动代理服务的各个组件
func (m *manager) Start() error { 
	if m.epl,err := MkEpoll() ;err!=nil{
				log.Error("[tcp_conn_handler] conn close!",
					zap.Error(err),
					zap.Any("username", proxyUserName),
					zap.Any("clientAddr", proxyServerIpStr),
					zap.Any("target_host", address),
				)
	} 
	m.initTcpListener()
	m.tcm.AddTask(AcceptAmount, m.tcpAccept)
	return nil
}

// Stop 停止所有服务组件
func (m *manager) Stop() {
	m.isRun.Store(false)
	for _, v := range m.tcpListener {
		v.Close()
	}
	m.tcm.Stop() // 停止任务消费者管理器，会触发所有任务的优雅关闭
}

func (m *manager) Valid(ctx context.Context, username, password, ip string) (authInfo *protobuf.AuthInfo, resErr error) {
 return nil
}