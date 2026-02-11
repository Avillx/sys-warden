package main

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type AlertMessage struct {
	Name    string
	details []string
}

func (o *WrappedObserver) SendAlert(d Details) {

	a := AlertMessage{
		Name:    o.Name,
		details: d,
	}

	body, _ := json.Marshal(a)

	req, _ := http.NewRequest("POST", "", bytes.NewBuffer(body))

	http.DefaultClient.Do(req)
}
