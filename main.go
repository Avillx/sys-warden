package main

import (
	"context"
	"fmt"
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

func main() {
	CreateContainerObserver()

	ctx := context.Background()
	for _ = range ctx.Done() {

	}
}
