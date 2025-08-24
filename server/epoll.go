package server

import (
	"net"
	"reflect"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

type epoll struct {
	fd          int
	connections map[int]net.Conn
	lock        *sync.RWMutex
	closeOne    func() error
}

func MkEpoll() (*epoll, error) {
	fd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}
	e := &epoll{
		fd:          fd,
		lock:        &sync.RWMutex{},
		connections: make(map[int]net.Conn),
	}
	e.closeOne = sync.OnceValue(func() error {
		e.lock.Lock()
		defer e.lock.Unlock()

		return unix.Close(fd)
	})

	return e, nil
}

func (e *epoll) Close() error {
	return e.closeOne()
}

func (e *epoll) Add(conn net.Conn, Pad int32) error {
	e.lock.Lock()
	defer e.lock.Unlock()
	// 提取连接的文件描述符
	fd := socketFD(conn)
	err := unix.EpollCtl(e.fd, syscall.EPOLL_CTL_ADD, fd,
		&unix.EpollEvent{Events: unix.POLLIN | unix.POLLHUP, Fd: int32(fd), Pad: Pad})
	if err != nil {
		return err
	}

	e.connections[fd] = conn
	return nil
}

func (e *epoll) Remove(conn net.Conn) error {
	e.lock.Lock()
	defer e.lock.Unlock()
	fd := socketFD(conn)
	err := unix.EpollCtl(e.fd, syscall.EPOLL_CTL_DEL, fd, nil)
	if err != nil {
		return err
	}

	delete(e.connections, fd)

	return nil
}

func (e *epoll) Wait() ([]unix.EpollEvent, error) {
	events := make([]unix.EpollEvent, 10)
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
