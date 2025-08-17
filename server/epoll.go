package server
import ( 
	"net" 
	"sync"
	"syscall" 
 
	"golang.org/x/sys/unix"
	"reflect"
)

type epoll struct {
	fd          int
	connections map[int]net.Conn
	lock        *sync.RWMutex
}
func MkEpoll( ) (*epoll, error) {
	fd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}
  
	return &epoll{
		fd:          fd,
		lock:        &sync.RWMutex{},
		connections: make(map[int]net.Conn), 
	}, nil
}


func (e *epoll) Close( ) error {
	return unix.Close(e.fd)
}

func (e *epoll) Add(conn net.Conn) error {
	
	// 提取连接的文件描述符
	fd := socketFD(conn)
	err := unix.EpollCtl(e.fd, syscall.EPOLL_CTL_ADD, fd, 
		&unix.EpollEvent{Events: unix.POLLIN | unix.POLLHUP, Fd: int32(fd)})
	if err != nil {
		return err
	}
	
	e.lock.Lock()
	defer e.lock.Unlock()
	e.connections[fd] = conn
	return nil
}

func (e *epoll) Remove(fd int) error {
	err := unix.EpollCtl(e.fd, syscall.EPOLL_CTL_DEL, fd, nil)
	if err != nil {
		return err
	}
	
	e.lock.Lock()
	defer e.lock.Unlock()
	delete(e.connections, fd)
 
	return nil
}

func (e *epoll) Wait() ([]unix.EpollEvent, error) {
	events := make([]unix.EpollEvent, 100)
retry:
	n, err := unix.EpollWait(e.fd, events, 100)
	if err != nil {
		if err == unix.EINTR {
			goto retry
		}
		return nil, err
	}
	return events[:n], nil
}

 
func socketFD(conn interface{}) int {
	val := reflect.Indirect(reflect.ValueOf(conn))

	// 处理连接类型
	tcpConn := val.FieldByName("conn")
	fdVal := tcpConn.FieldByName("fd")
	pfdVal := reflect.Indirect(fdVal).FieldByName("pfd")
	return int(pfdVal.FieldByName("Sysfd").Int())
}