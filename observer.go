package main

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

type FileUpdates UpdatesChannel
type UpdatesChannel chan []string
type FilePath string
type Details []string
type OnInvokeCallback func(d Details)

type WrappedObserver struct {
	Name     string
	Observer Observer
	OnInvoke OnInvokeCallback
	Cancel   context.CancelFunc
}

func (w *WrappedObserver) Run(ctx context.Context) {

	context.AfterFunc(ctx, func() {

		w.Observer.Release()
	})

	go func() {
		for {
			if ctx.Err() == context.Canceled {
				return
			}

			w.Observer.AwaitInvoke()

		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case u := <-w.Observer.GetUpdateChan():
				w.OnInvoke(u)
			}
		}
	}()
}

type Observer interface {
	AwaitInvoke()
	GetUpdateChan() UpdatesChannel
	Release()
}

type FileObserver struct {
	fileDescriptiorID  int
	watchDescriptiorID int
	file               *os.File
	offset             int64
	buffer             []byte
	updateChan         FileUpdates
}

func NewFileObserver(f FilePath) (*FileObserver, error) {

	fd, err := syscall.InotifyInit()
	if err != nil {
		return &FileObserver{}, err
	}

	file, err := os.Open(string(f))
	if err != nil {
		return &FileObserver{}, err
	}

	wd, err := syscall.InotifyAddWatch(fd, string(f), syscall.IN_MODIFY)
	if err != nil {
		return &FileObserver{}, err
	}

	stat, _ := file.Stat()

	return &FileObserver{
		fileDescriptiorID:  fd,
		watchDescriptiorID: wd,
		file:               file,
		offset:             stat.Size(),
		buffer:             make([]byte, 4096),
		updateChan:         make(chan []string),
	}, nil
}

func (o *FileObserver) AwaitInvoke() {
	n, err := syscall.Read(o.fileDescriptiorID, o.buffer)

	if err != nil {
		fmt.Errorf(err.Error())
	}

	var pos uint32
	for pos < uint32(n) {
		event := (*syscall.InotifyEvent)(unsafe.Pointer(&o.buffer[pos]))

		newContentBuf := make([]byte, 1024)
		bytesRead, _ := o.file.ReadAt(newContentBuf, o.offset)
		if bytesRead == 0 {

			return
		}

		o.updateChan <- []string{string(newContentBuf[:bytesRead])}

		o.offset += int64(bytesRead)

		pos += syscall.SizeofInotifyEvent + event.Len
	}
}

func (o *FileObserver) GetUpdateChan() UpdatesChannel {

	return UpdatesChannel(o.updateChan)
}

func (o *FileObserver) Release() {
	syscall.InotifyRmWatch(o.fileDescriptiorID, uint32(o.watchDescriptiorID))
	syscall.Close(o.fileDescriptiorID)
}
