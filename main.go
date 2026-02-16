package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

func CreateTempoFileObserver() {
	o, err := NewFileObserver("./tmp.txt")
	if err != nil {
		fmt.Printf("obs creation err " + err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())

	wo := WrappedObserver{
		Name:     "Tempo",
		Observer: o,
		OnUpdate: func(d Update) {
			fmt.Print(d)
		},
		Cancel: cancel,
	}

	wo.Run(ctx)
}

func CreateContainerObserver() {
	o, err := NewContainerObserver("b5b67974e4a7c40206e50778a8f6885438e952e67f04633aacdc29eb183eda62")
	if err != nil {
		fmt.Print("obs creation err " + err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())

	wo := WrappedObserver{
		Name:     "Container",
		Observer: o,
		OnUpdate: func(d Update) {
			fmt.Print(d)
		},
		Cancel: cancel,
	}

	wo.Run(ctx)
}

func getInfo() {
	provider := NewDockerProvider()

	var reader io.ReadCloser

	_ = provider.ExecuteDockerRequest(
		NewGetContainerOutputRequest(&reader, "223ca1b1ff3d1d474edb0938007ed32b9de37d67236b10aff148b0e895bf0b4e"),
	)

	go func() {
		for {
			header := make([]byte, 8)

			_, err := io.ReadFull(reader, header)

			if err != nil {
				fmt.Printf(err.Error())
				continue
			}

			frameSize := binary.BigEndian.Uint32(header[4:])
			content := make([]byte, frameSize)

			_, err = io.ReadFull(reader, content)

			if err != nil {
				fmt.Printf(err.Error())
			}

			if len(content) <= 0 {

				continue
			}

			fmt.Printf(string(content))
			time.Sleep(1 * time.Second)
		}
	}()

}

func main() {
	getInfo()

	ctx := context.Background()
	for _ = range ctx.Done() {

	}
}
