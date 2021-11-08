package bootstrap

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
)

type HttpEndpoint struct {
	Storage Storage
	Router  *httprouter.Router

	EnableAuth     bool
	AccessPassword string

	Addr string

	DebugPrint bool

	publicClients map[string]*struct {
		LastUpdate  int64
		Publishment map[string]struct {
			Addr string
		}
	}
	rwlock sync.RWMutex
}

func NewHttpEndpoint(addr string, storage Storage, enableAuth bool, accessPassword string) (*HttpEndpoint, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	r := new(HttpEndpoint)
	r.Storage = storage
	r.Addr = u.Host
	r.EnableAuth = enableAuth
	r.AccessPassword = accessPassword

	router := httprouter.New()
	router.GET("/service", r.queryService)
	router.POST("/service", r.publishService)
	r.Router = router

	r.publicClients = make(map[string]*struct {
		LastUpdate  int64
		Publishment map[string]struct {
			Addr string
		}
	})

	return r, nil
}

func (this *HttpEndpoint) StartSync() error {
	// cleanup timeout
	go this.CheckPublishClients()

	return http.ListenAndServe(this.Addr, this.Router)
}

func (this *HttpEndpoint) CheckPublishClients() {
	// cleanup timeout
	for {
		const interval = 10
		time.Sleep(1 * time.Second)

		f := func() {
			this.rwlock.Lock()
			defer this.rwlock.Unlock()

			barrierTime := time.Now().UnixMilli() - interval*1000
			var rlist []string
			for k, v := range this.publicClients {
				if v.LastUpdate < barrierTime {
					rlist = append(rlist, k)
				}
			}
			for _, vv := range rlist {
				p := this.publicClients[vv]
				for serviceName, v := range p.Publishment {
					err := this.Storage.DeleteService(serviceName, &ServiceItem{
						Addr:   v.Addr,
						NodeId: vv,
					})
					if err != nil {
						log.Println("[ERROR] cleanup service error:", err)
					}
				}
				delete(this.publicClients, vv)
			}
		}
		f()
	}
}

func (this *HttpEndpoint) queryService(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	_ = r.ParseForm()
	serviceName := r.FormValue("name")

	if this.EnableAuth {
		accessPassword := r.Header.Get("X-Access-Password")
		if accessPassword != this.AccessPassword {
			log.Println("[ERROR] access password doesn't match")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	if this.DebugPrint {
		log.Println("[DEBUG] query service:", serviceName)
	}

	w.Header().Set("Content-Type", "application/json")

	items, err := this.Storage.GetServiceList(serviceName)
	if err != nil {
		log.Println("[ERROR] get service list error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(items) == 0 {
		items = []*ServiceItem{}
	}
	data, err := json.Marshal(&items)
	if err != nil {
		log.Println("[ERROR] marshal result error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(data); err != nil {
		log.Println("[ERROR] queryService response fail:", err)
	}
}

func (this *HttpEndpoint) publishService(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	_ = r.ParseForm()
	serviceName := r.FormValue("name")
	nodeId := r.FormValue("node")
	address := r.FormValue("address")

	if this.EnableAuth {
		accessPassword := r.Header.Get("X-Access-Password")
		if accessPassword != this.AccessPassword {
			log.Println("[ERROR] access password doesn't match")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	if this.DebugPrint {
		log.Println("[DEBUG] publish service:", serviceName, nodeId, address)
	}

	w.Header().Set("Content-Type", "application/json")

	err := this.Storage.PublishService(serviceName, &ServiceItem{
		Addr:   address,
		NodeId: nodeId,
	})
	if err != nil {
		log.Println("[ERROR] publish service error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// publish list
	f := func() {
		this.rwlock.Lock()
		defer this.rwlock.Unlock()
		p, ok := this.publicClients[nodeId]
		if !ok {
			p = &struct {
				LastUpdate  int64
				Publishment map[string]struct{ Addr string }
			}{LastUpdate: time.Now().UnixMilli(), Publishment: map[string]struct{ Addr string }{}}
			this.publicClients[nodeId] = p
		}
		p.LastUpdate = time.Now().UnixMilli()
		p.Publishment[serviceName] = struct{ Addr string }{Addr: address}
	}
	f()

	w.WriteHeader(http.StatusOK)
}
