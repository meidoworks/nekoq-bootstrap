package service

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"

	"github.com/meidoworks/nekoq-bootstrap/internal/httpserver"
	"github.com/meidoworks/nekoq-bootstrap/internal/iface"
)

type HttpServiceContainer struct {
	h *httpserver.HttpServer

	kvstor iface.KVStorage
}

func NewHttpServiceContainer(h *httpserver.HttpServer) *HttpServiceContainer {
	return &HttpServiceContainer{
		h: h,
	}
}

func (h *HttpServiceContainer) SetupKVStorage(stor iface.KVStorage) *HttpServiceContainer {
	h.kvstor = stor
	h.h.Add(httpserver.MethodGet, "/services/kvstore/:key", func(request *http.Request, params httprouter.Params) (iface.HttpResult, error) {
		key := strings.TrimSpace(params.ByName("key"))
		if key == "" {
			return iface.StatusOnlyResult(http.StatusBadRequest), nil
		}
		val, exist, err := h.kvstor.Get([]byte(key))
		if err != nil {
			log.Println("ERROR: http kvstore get error:", err)
			return nil, err
		}
		if !exist {
			return iface.StatusOnlyResult(http.StatusNotFound), nil
		}
		return iface.RawResult(val), nil
	})
	h.h.Add(httpserver.MethodPut, "/services/kvstore/:key", func(request *http.Request, params httprouter.Params) (iface.HttpResult, error) {
		key := strings.TrimSpace(params.ByName("key"))
		if key == "" {
			return iface.StatusOnlyResult(http.StatusBadRequest), nil
		}
		dat, err := io.ReadAll(request.Body)
		if err != nil {
			log.Println("ERROR: http kvstore put -> read http body error:", err)
			return nil, err
		}
		if err := h.kvstor.Put([]byte(key), dat); err != nil {
			log.Println("ERROR: http kvstore put -> persist kv error:", err)
		}
		return iface.StatusOnlyResult(http.StatusOK), nil
	})
	return h
}

func (h *HttpServiceContainer) Startup() error {
	return h.h.Startup()
}

func (h *HttpServiceContainer) Stop() error {
	return h.h.Stop()
}
