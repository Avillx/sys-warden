package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
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

// ////////////////////////////////////////////////////
//

type ContainerObserver struct {
	ContainerID  string
	SocketClient http.Client
	SocketReader io.ReadCloser
}

func NewContainerObserver(containerID string) (*ContainerObserver, error) {
	socketClient := getSocketClient()
	dockerVersion, err := getDockerVersion(socketClient)

	if err != nil {
		return &ContainerObserver{}, err
	}
	socketReader, err := getContainerReader(containerID, dockerVersion, socketClient)

	if err != nil {
		return &ContainerObserver{}, err
	}

	return &ContainerObserver{
		ContainerID:  containerID,
		SocketClient: socketClient,
		SocketReader: socketReader,
	}, nil
}

func (o *ContainerObserver) AwaitUpdate() Update {
	readerx, _ := getContainerReader(o.ContainerID, "28.5.2", o.SocketClient)
	header := make([]byte, 8)

	_, err := io.ReadFull(readerx, header)
	if err != nil {
		return nil //err // Вернет ошибку при разрыве соединения (EOF и др.)
	}

	frameSize := binary.BigEndian.Uint32(header[4:])
	content := make([]byte, frameSize)

	_, err = io.ReadFull(readerx, content)
	// if err != nil {
	// 	return nil
	// }

	if len(content) <= 0 {

		return nil
	}

	return Update{string(content)}
}

func (o *ContainerObserver) Release() {
	// must unlock io.ReadAll
	o.SocketReader.Close()
}

func getSocketClient() http.Client {
	socketPath := "/var/run/docker.sock"
	return http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}
}

type dockerVersion struct {
	Version string `json:"Version"`
}

func getDockerVersion(SocketClient http.Client) (string, error) {
	resp, err := SocketClient.Get("http://localhost/version")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", err
	}

	var v dockerVersion
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return "", err
	}

	return v.Version, nil
}

func getContainerReader(containerID, dockerVersion string, socketClient http.Client) (io.ReadCloser, error) {

	url := fmt.Sprintf("http://localhost/%s/containers/%s/logs?stdout=1&stderr=1&follow=1", dockerVersion, containerID)

	resp, err := socketClient.Get(url)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}
