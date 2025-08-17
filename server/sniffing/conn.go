package sniffing

import (
	"fmt"
	"io"
	"slices"
	"sync"
	"sync/atomic"
)

type ReadWriterNotice struct {
	rw       io.ReadWriteCloser
	buf      []byte
	l        sync.Mutex
	isNotice atomic.Bool
	isStart  atomic.Bool
	fc       func(buf []byte)
}

func NewReadWriterNotice(
	rw io.ReadWriteCloser,
	buf []byte,
	fc func(buf []byte),
) (*ReadWriterNotice, error) {
	if rw == nil {
		return nil, fmt.Errorf("rw io.ReadWriter  == nil")
	}
	buf = slices.Clone(buf)
	rwn := &ReadWriterNotice{
		rw:  rw,
		buf: buf,
		fc:  fc,
	}

	rwn.isNotice.Swap(false)
	rwn.isStart.Swap(false)
	if rwn.buf == nil {
		rwn.buf = []byte{}
	}

	rwn.isStart.Swap(true)
	return rwn, nil
}

func (rwn *ReadWriterNotice) Close() error {
	if !rwn.isStart.CompareAndSwap(true, false) {
		return fmt.Errorf("ReadWriterNotice is not start")
	}

	rwn.l.Lock()
	rwn.buf = nil
	rwn.l.Unlock()

	return rwn.rw.Close()
}

func (rwn *ReadWriterNotice) Read(p []byte) (n int, err error) {
	if !rwn.isStart.Load() {
		return 0, io.EOF
	}
	n, err = rwn.rw.Read(p)

	if !rwn.isNotice.Load() {
		rwn.l.Lock()
		defer rwn.l.Unlock()

		if rwn.buf != nil {
			rwn.buf = append(rwn.buf, p[:n]...)
		}
	}
	return
}

func (rwn *ReadWriterNotice) Write(p []byte) (n int, err error) {
	if !rwn.isStart.Load() {
		return 0, io.EOF
	}

	n, err = rwn.rw.Write(p)
	if rwn.isNotice.CompareAndSwap(false, true) {
		if rwn.fc == nil {
			return
		}

		rwn.l.Lock()
		if rwn.buf == nil {
			rwn.l.Unlock()
			return
		}
		buf := rwn.buf
		rwn.buf = nil
		rwn.l.Unlock()
		rwn.fc(buf)

	}
	return
}
