package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
)

var DefaultSocketPath = "/var/run/docker.sock"
var DockerAPIVerison = ""

type ContainerInfoStruct struct {
	ID   string
	Name string
}

type ContainerRecord struct {
	Id   string
	Name string
}

type DockerProvider struct {
	SocketPath   string
	SocketClient http.Client
	APIVersion   string
}

func NewDockerProvider() *DockerProvider {
	socketPath := DefaultSocketPath
	socketClient := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}

	dockerProvider := &DockerProvider{
		SocketPath:   socketPath,
		SocketClient: socketClient,
	}

	var dockerInfo DockerVersionInfo
	// execute request without version is unsafe but this counstructor
	// guarantee that returned provider has version
	err := dockerProvider.ExecuteDockerRequest(
		NewGetDockerInfoRequest(&dockerInfo),
	)

	if err != nil {
		panic("Docker is unreachible")
	}

	//TODO:
	// check version compability
	DockerAPIVerison = dockerInfo.Components[0].Details.ApiVersion
	dockerProvider.APIVersion = DockerAPIVerison

	return dockerProvider
}

func (d *DockerProvider) ExecuteDockerRequest(r Requestable) error {

	ver := ""
	if r.IsVersionRequired() {
		ver = fmt.Sprintf("/v%s", d.APIVersion)
	}

	query := ""
	if r.Query() != "" {
		query = fmt.Sprintf("?%s", r.Query())
	}

	url := fmt.Sprintf("http://localhost%s%s%s", ver, r.EndPoint(), query)

	socketReqest, err := http.NewRequest(r.Method(), url, bytes.NewBuffer([]byte{}))

	if err != nil {

		return err
	}

	resp, err := d.SocketClient.Do(socketReqest)
	if err != nil {

		return err
	}

	return r.Resolve(resp)
}

type Requestable interface {
	Query() string
	Method() string
	EndPoint() string
	IsVersionRequired() bool
	Resolve(resp *http.Response) error
}

type RequestResolver[T any] func(resp *http.Response, resultPtr *T) error

type DockerRequest[T any] struct {
	endPoint    string
	method      string
	query       string
	verRequired bool
	resolver    RequestResolver[T]
	resultPtr   *T
}

func (r *DockerRequest[T]) Resolve(resp *http.Response) error {
	return r.resolver(resp, r.resultPtr)
}

func (r *DockerRequest[T]) IsVersionRequired() bool { return r.verRequired }

func (r *DockerRequest[T]) EndPoint() string { return r.endPoint }

func (r *DockerRequest[T]) Method() string { return r.method }

func (r *DockerRequest[T]) Query() string { return r.query }

// req
//
// req
type GetDockerInfoRequest struct {
	DockerRequest[DockerVersionInfo]
}

func NewGetDockerInfoRequest(result *DockerVersionInfo) *GetDockerInfoRequest {
	resolverFunc := func(resp *http.Response, resultPtr *DockerVersionInfo) error {
		data, err := io.ReadAll(resp.Body)

		if err != nil {
			return err
		}

		return json.Unmarshal(data, resultPtr)
	}

	return &GetDockerInfoRequest{
		DockerRequest: DockerRequest[DockerVersionInfo]{
			endPoint:    "/version",
			method:      "GET",
			query:       "",
			verRequired: false,
			resultPtr:   result,
			resolver:    resolverFunc,
		},
	}
}

// req
//
// req
type GetContainersListRequest struct {
	DockerRequest[[]DockerContainer]
}

func NewGetContainersListRequest(result *[]DockerContainer) *GetContainersListRequest {

	resolverFunc := func(resp *http.Response, resultPtr *[]DockerContainer) error {
		data, err := io.ReadAll(resp.Body)

		if err != nil {
			return err
		}

		return json.Unmarshal(data, resultPtr)
	}

	return &GetContainersListRequest{
		DockerRequest: DockerRequest[[]DockerContainer]{
			endPoint:    "/containers/json",
			method:      "GET",
			query:       "",
			verRequired: true,
			resultPtr:   result,
			resolver:    resolverFunc,
		},
	}
}

// req
//
// req
type GetContainerOutputRequest struct {
	DockerRequest[io.ReadCloser]
}

func NewGetContainerOutputRequest(result *io.ReadCloser, ContainerID string) *GetContainerOutputRequest {

	resolverFunc := func(resp *http.Response, resultPtr *io.ReadCloser) error {

		if resp.StatusCode != 200 {
			// TODO Exclude to typed errors
			data, _ := io.ReadAll(resp.Body)
			errorMessage := fmt.Sprintf("GetContianerOutputerror: %s", string(data))
			return errors.New(errorMessage)
		}

		*resultPtr = resp.Body
		return nil
	}

	endPoint := fmt.Sprintf("/containers/%s/logs", ContainerID)

	return &GetContainerOutputRequest{

		DockerRequest: DockerRequest[io.ReadCloser]{
			endPoint:    endPoint,
			method:      "GET",
			query:       "stdout=1&stderr=1&follow=1",
			verRequired: true,
			resultPtr:   result,
			resolver:    resolverFunc,
		},
	}
}
