package service

import (
	"log"

	"github.com/meidoworks/nekoq-bootstrap/internal/iface"
)

type RespKVHandler struct {
	kv iface.KVStorageAdv
}

func NewRespKVHandler(kv iface.KVStorageAdv) RespKVHandler {
	return RespKVHandler{
		kv: kv,
	}
}

func (h RespKVHandler) Register(rs *Resp2Service) {
	rs.AddCommandHandler("get", h.get)
	rs.AddCommandHandler("put", h.put)
}

func (r RespKVHandler) get(args []iface.RespArg) (iface.RespResult, error) {
	if len(args) < 2 {
		return iface.RespErrorResult("lack arguments"), nil
	}
	val, found, err := r.kv.Get(args[1])
	if err != nil {
		log.Println("RespKVHandler get failed:", err)
		return iface.RespErrorResult("Get value failed."), nil
	}
	if !found {
		return iface.RespNilResult(), nil
	} else {
		return iface.RespValueResult(val), nil
	}
}

func (r RespKVHandler) put(args []iface.RespArg) (iface.RespResult, error) {
	if len(args) < 3 {
		return iface.RespErrorResult("lack arguments"), nil
	}
	if err := r.kv.Put(args[1], args[2]); err != nil {
		log.Println("RespKVHandler put failed:", err)
		return iface.RespErrorResult("Put value failed"), nil
	} else {
		return iface.RespStringResult("OK"), nil
	}
}
