package main

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

type FilePath string
type Details []string
type Update []string
type OnUpdateCallback func(u Update)

type WrappedObserver struct {
	Name     string
	Observer Observer
	OnUpdate OnUpdateCallback
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

			update := w.Observer.AwaitUpdate()

			if len(update) > 0 {
				w.OnUpdate(update)
			}
		}
	}()
}

type Observer interface {
	AwaitUpdate() Update
	Release()
}

type FileObserver struct {
	fileDescriptiorID  int
	watchDescriptiorID int
	file               *os.File
	offset             int64
	buffer             []byte
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
	}, nil
}

func (o *FileObserver) AwaitUpdate() Update {
	n, err := syscall.Read(o.fileDescriptiorID, o.buffer)

	if err != nil {
		fmt.Errorf(err.Error())
	}
	update := Update{}

	var pos uint32
	for pos < uint32(n) {
		event := (*syscall.InotifyEvent)(unsafe.Pointer(&o.buffer[pos]))

		newContentBuf := make([]byte, 1024)
		bytesRead, _ := o.file.ReadAt(newContentBuf, o.offset)
		if bytesRead == 0 {

			continue
		}

		update = append(update, string(newContentBuf[:bytesRead]))

		o.offset += int64(bytesRead)

		pos += syscall.SizeofInotifyEvent + event.Len
	}

	return update
}

func (o *FileObserver) Release() {
	syscall.InotifyRmWatch(o.fileDescriptiorID, uint32(o.watchDescriptiorID))
	syscall.Close(o.fileDescriptiorID)
}
