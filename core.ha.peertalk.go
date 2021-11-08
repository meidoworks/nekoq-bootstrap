package bootstrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/julienschmidt/httprouter"
	"github.com/miekg/dns"
)

const (
	_HttpAuth        = "/cluster/auth"
	_HttpFull        = "/cluster/full"
	_HttpIncremental = "/cluster/incremental"
)

func (this *HaModule) innerUrlResolve(addr string) string {
	u, err := url.Parse(addr)
	if err != nil {
		panic(err)
	}

	r, err := this.Storage.ResolveDomain(dns.Fqdn(u.Hostname()), DomainTypeA)
	if err == nil {
		u.Host = fmt.Sprint(r, ":", u.Port())
	}

	return u.String()
}

type conn struct {
	client *resty.Client
}

func (this *HaModule) updateClientTime(nodeId string, failure bool) error {
	this.ClientPeerMappingLock.Lock()
	defer this.ClientPeerMappingLock.Unlock()
	p, ok := this.ClientPeerMapping[nodeId]
	if !ok {
		if failure {
			return errors.New("node not exist")
		}
		p = &struct {
			LastUpdate int64
		}{LastUpdate: 0}
		this.ClientPeerMapping[nodeId] = p
	}
	p.LastUpdate = time.Now().UnixMilli()
	return nil
}

func (this *HaModule) connectRemote(host string) (*conn, error) {
	c := new(conn)
	c.client = resty.New()

	resp, err := c.client.R().
		SetHeaders(map[string]string{
			"X-Cluster-Name":   this.ClusterName,
			"X-Cluster-Secret": this.ClusterSecret,
			"Accept":           "application/json",
			"Content-Type":     "application/json",
		}).
		SetQueryParam("from", this.NodeId).
		Post(this.innerUrlResolve(host) + _HttpAuth)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, errors.New("cluster auth failed")
	}

	return c, nil
}
func (this *HaModule) httpClusterAuth(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	_ = r.ParseForm()
	nodeId := r.FormValue("from")
	name := r.Header.Get("X-Cluster-Name")
	secret := r.Header.Get("X-Cluster-Secret")

	if nodeId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := this.updateClientTime(nodeId, false); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if this.DebugPrint {
		log.Println("[DEBUG] httpClusterAuth node:", nodeId, "cluster_name:", name)
	}

	w.Header().Set("Content-Type", "application/json")

	if name != this.ClusterName || secret != this.ClusterSecret {
		log.Println("[ERROR] cluster name or cluster secret doesn't match")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (this *HaModule) queryFullDataSet(addr string, c *conn) (map[string][]byte, error) {
	resp, err := c.client.R().
		SetHeaders(map[string]string{
			"X-Cluster-Name":   this.ClusterName,
			"X-Cluster-Secret": this.ClusterSecret,
			"Accept":           "application/json",
			"Content-Type":     "application/json",
		}).
		SetQueryParam("from", this.NodeId).
		Get(this.innerUrlResolve(addr) + _HttpFull)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, errors.New("cluster full failed")
	}

	data := resp.Body()
	var dataMap = make(map[string][]byte)
	if err := json.Unmarshal(data, &dataMap); err != nil {
		return nil, err
	}

	return dataMap, nil
}
func (this *HaModule) httpClusterFull(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	_ = r.ParseForm()
	nodeId := r.FormValue("from")
	name := r.Header.Get("X-Cluster-Name")
	secret := r.Header.Get("X-Cluster-Secret")

	if nodeId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := this.updateClientTime(nodeId, false); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if this.DebugPrint {
		log.Println("[DEBUG] httpClusterFull node:", nodeId, "cluster_name:", name)
	}

	w.Header().Set("Content-Type", "application/json")

	if name != this.ClusterName || secret != this.ClusterSecret {
		log.Println("[ERROR] cluster name or cluster secret doesn't match")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	dat, err := this.Storage.FetchFullAndWatch(nodeId)
	if err != nil {
		log.Println("[ERROR] httpClusterFull failed:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(dat)
	if err != nil {
		log.Println("[ERROR] httpClusterFull json marshal failed:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(data)
	if err != nil {
		log.Println("[ERROR] httpClusterFull response failed:", err)
	}
}

func (this *HaModule) queryIncrementalData(addr string, c *conn) (map[string][]byte, map[string][]byte, error) {
	resp, err := c.client.R().
		SetHeaders(map[string]string{
			"X-Cluster-Name":   this.ClusterName,
			"X-Cluster-Secret": this.ClusterSecret,
			"Accept":           "application/json",
			"Content-Type":     "application/json",
		}).
		SetQueryParam("from", this.NodeId).
		Get(this.innerUrlResolve(addr) + _HttpIncremental)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, nil, errors.New("cluster incremental failed")
	}

	data := resp.Body()
	var dataMap = &struct {
		Add map[string][]byte `json:"add"`
		Del map[string][]byte `json:"del"`
	}{}
	if err := json.Unmarshal(data, dataMap); err != nil {
		return nil, nil, err
	}

	return dataMap.Add, dataMap.Del, nil
}
func (this *HaModule) httpClusterIncremental(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	_ = r.ParseForm()
	nodeId := r.FormValue("from")
	name := r.Header.Get("X-Cluster-Name")
	secret := r.Header.Get("X-Cluster-Secret")

	if nodeId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := this.updateClientTime(nodeId, true); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if this.DebugPrint {
		log.Println("[DEBUG] httpClusterIncremental node:", nodeId, "cluster_name:", name)
	}

	w.Header().Set("Content-Type", "application/json")

	if name != this.ClusterName || secret != this.ClusterSecret {
		log.Println("[ERROR] cluster name or cluster secret doesn't match")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	add, del, err := this.Storage.FetchChangesForPeerNodeRequest(nodeId)
	if err != nil {
		log.Println("[ERROR] httpClusterIncremental failed:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var dat = &struct {
		Add map[string][]byte `json:"add"`
		Del map[string][]byte `json:"del"`
	}{Add: add, Del: del}

	data, err := json.Marshal(dat)
	if err != nil {
		log.Println("[ERROR] httpClusterIncremental json marshal failed:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(data)
	if err != nil {
		log.Println("[ERROR] httpClusterIncremental response failed:", err)
	}
}

func (this *HaModule) HttpEndpoint() error {
	router := httprouter.New()
	router.POST(_HttpAuth, this.httpClusterAuth)
	router.GET(_HttpFull, this.httpClusterFull)
	router.GET(_HttpIncremental, this.httpClusterIncremental)

	u, err := url.Parse(this.Listen)
	if err != nil {
		panic(err)
	}

	this.router = router

	err = http.ListenAndServe(u.Host, this)
	if err != nil {
		panic(err)
	}

	return nil
}
func (this *HaModule) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if this.DebugPrint {
		log.Println("[DEBUG] access url:", r.URL.String(), "method:", r.Method)
	}
	this.router.ServeHTTP(w, r)
}
