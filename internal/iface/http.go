package iface

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type HttpHandler func(request *http.Request, params httprouter.Params) (HttpResult, error)

type HttpResultRender func(w http.ResponseWriter) error

type HttpResult interface {
	Render(w http.ResponseWriter) error
}

type wrappedResultRender struct {
	render HttpResultRender
}

func (wr wrappedResultRender) Render(w http.ResponseWriter) error {
	return wr.render(w)
}

func WrapResultRender(render HttpResultRender) HttpResult {
	return wrappedResultRender{render: render}
}

func JsonResult(status int, object any) HttpResult {
	return WrapResultRender(func(w http.ResponseWriter) error {
		data, err := json.Marshal(object)
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		_, err = w.Write(data)
		return err
	})
}

func StatusOnlyResult(status int) HttpResult {
	return WrapResultRender(func(w http.ResponseWriter) error {
		w.WriteHeader(status)
		return nil
	})
}

func RawResult(raw []byte) HttpResult {
	return WrapResultRender(func(w http.ResponseWriter) error {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write(raw)
		return err
	})
}
