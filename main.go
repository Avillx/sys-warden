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

func main() {
	CreateTempoFileObserver()

	ctx := context.Background()
	for _ = range ctx.Done() {

	}
}
