package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

type DockerRequest interface {
	endPoint() string
	method() string
	query() string
}

type DockerProvider struct {
	SocketPath   string
	SocketClient http.Client
	DataReader   io.ReadCloser
}

func (d *DockerProvider) ExecuteRequest(r DockerRequest) {

	url := fmt.Sprintf("http://%s/%s", r.endPoint(), r.query())

	socketReqest, err := http.NewRequest(r.method(), url, bytes.NewBuffer([]byte{}))

	if err != nil {

		return _, err
	}

	socketReqest
	//d.SocketClient.Do()
}
