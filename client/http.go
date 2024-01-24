package client

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type HttpServiceClient struct {
	endpoints []string
	client    *http.Client
}

func NewHttpServiceClient(endpoint string) *HttpServiceClient {
	return &HttpServiceClient{
		endpoints: []string{endpoint},
		client:    &http.Client{},
	}
}

func (h *HttpServiceClient) KvPut(key string, val []byte) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("key is empty")
	}

	r, err := http.NewRequest(http.MethodPut, h.pickEndpoint()+fmt.Sprint("/services/kvstore/", key), bytes.NewReader(val))
	if err != nil {
		return err
	}
	resp, err := h.client.Do(r)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	default:
		return errors.New(fmt.Sprint("put kv failed, status:", resp.StatusCode))
	}
}

func (h *HttpServiceClient) KvGet(key string) ([]byte, bool, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, false, errors.New("key is empty")
	}

	r, err := h.client.Get(h.pickEndpoint() + fmt.Sprint("/services/kvstore/", key))
	if err != nil {
		return nil, false, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(r.Body)
	switch r.StatusCode {
	case http.StatusNotFound:
		return nil, false, nil
	case http.StatusOK:
		if dat, err := io.ReadAll(r.Body); err != nil {
			return nil, false, err
		} else {
			return dat, true, nil
		}
	default:
		return nil, false, errors.New(fmt.Sprint("get kv failed, status:", r.StatusCode))
	}
}

func (h *HttpServiceClient) pickEndpoint() string {
	return h.endpoints[0]
}
