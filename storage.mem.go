package bootstrap

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/miekg/dns"

	"github.com/meidoworks/nekoq-bootstrap/internal/shared"
)

const (
	_DefaultDataKey = "__default__"
)

type MemStore struct {
	domains  map[string]string
	services map[string]map[string]struct {
		Addr string
	}

	// current node data
	currentServices map[string]map[string]struct {
		Addr string
	}
	// received from remote
	dataFrom map[string]*memNodeData
	// pending sending data to remote
	dataTo map[string]*struct {
		Add []*dataTransferNodeData
		Del []*dataTransferNodeData
	}

	rwlock sync.RWMutex
}

func (this *MemStore) GetServiceList(service string) ([]*ServiceItem, error) {
	this.rwlock.RLock()
	defer this.rwlock.RUnlock()

	li, ok := this.services[service]
	if !ok {
		return nil, nil
	}

	var r []*ServiceItem
	for k, v := range li {
		r = append(r, &ServiceItem{
			Addr:   v.Addr,
			NodeId: k,
		})
	}

	return r, nil
}

func (this *MemStore) PublishService(service string, item *ServiceItem) error {
	this.rwlock.Lock()
	defer this.rwlock.Unlock()

	// update storage
	{
		p, ok := this.services[service]
		if !ok {
			p = map[string]struct{ Addr string }{}
			this.services[service] = p
		}
		p[item.NodeId] = struct{ Addr string }{Addr: item.Addr}
	}
	// update current
	{
		p, ok := this.currentServices[service]
		if !ok {
			p = map[string]struct{ Addr string }{}
			this.currentServices[service] = p
		}
		p[item.NodeId] = struct{ Addr string }{Addr: item.Addr}
	}
	// update pending list
	{
		for _, v := range this.dataTo {
			v.Del = rebuildServiceItemList(v.Del, service, item)
			v.Add = append(v.Add, &dataTransferNodeData{
				ServiceName: service,
				NodeName:    item.NodeId,
				Info: struct {
					Addr string
				}{Addr: item.Addr},
			})
		}
	}

	return nil
}

func rebuildServiceItemList(del []*dataTransferNodeData, service string, item *ServiceItem) []*dataTransferNodeData {
	newList := make([]*dataTransferNodeData, 0, len(del))
	for _, v := range del {
		if v.ServiceName == service && v.NodeName == item.NodeId {
			continue
		}
		newList = append(newList, v)
	}
	return newList
}

func (this *MemStore) DeleteService(service string, item *ServiceItem) error {
	this.rwlock.Lock()
	defer this.rwlock.Unlock()

	// update storage
	{
		p, ok := this.services[service]
		if ok {
			delete(p, item.NodeId)
			if len(p) == 0 {
				delete(this.services, service)
			}
		}
	}
	// update current
	{
		p, ok := this.currentServices[service]
		if ok {
			delete(p, item.NodeId)
			if len(p) == 0 {
				delete(this.currentServices, service)
			}
		}
	}
	// update pending list
	{
		for _, v := range this.dataTo {
			v.Del = append(v.Del, &dataTransferNodeData{
				ServiceName: service,
				NodeName:    item.NodeId,
				Info: struct {
					Addr string
				}{Addr: item.Addr},
			})
			v.Add = rebuildServiceItemList(v.Add, service, item)
		}
	}

	return nil
}

func (this *MemStore) buildMemData() {
	ch := make(chan map[string]map[string]struct {
		Addr string
	}, 1)
	go func() {
		newServiceMap := make(map[string]map[string]struct {
			Addr string
		})
		copyMap(this.currentServices, newServiceMap)
		for _, v := range this.dataFrom {
			copyMap(v.ServiceMapping, newServiceMap)
		}
		ch <- newServiceMap
	}()
	this.services = <-ch
}

func copyMap(src, dst map[string]map[string]struct {
	Addr string
}) {
	for k, v := range src {
		sub, ok := dst[k]
		if !ok {
			sub = map[string]struct{ Addr string }{}
			dst[k] = sub
		}
		for i, j := range v {
			sub[i] = j
		}
	}
}

type memNodeData struct {
	ServiceMapping map[string]map[string]struct {
		Addr string
	}
}

type dataTransferNodeData struct {
	ServiceName string
	NodeName    string
	Info        struct {
		Addr string
	}
}

func memNodeDataUnmarshall(data []byte) (*memNodeData, error) {
	m := new(memNodeData)
	err := json.Unmarshal(data, m)
	return m, err
}

func dataTransferNodeDataList(data []byte) ([]*dataTransferNodeData, error) {
	var l []*dataTransferNodeData
	err := json.Unmarshal(data, &l)
	return l, err
}

func (m *MemStore) FullFrom(node string, data map[string][]byte) error {
	d, ok := data[_DefaultDataKey]
	if !ok {
		return errors.New("no data found")
	}
	dat, err := memNodeDataUnmarshall(d)
	if err != nil {
		return err
	}

	m.rwlock.Lock()
	defer m.rwlock.Unlock()

	m.dataFrom[node] = dat
	m.buildMemData()
	return nil
}

func (m *MemStore) SyncFrom(node, origNode string, add, del map[string][]byte) error {
	ad, ok := add[_DefaultDataKey]
	if !ok {
		return errors.New("no add data found")
	}
	adata, err := dataTransferNodeDataList(ad)
	if err != nil {
		return err
	}
	dd, ok := del[_DefaultDataKey]
	if !ok {
		return errors.New("no del data found")
	}
	ddata, err := dataTransferNodeDataList(dd)
	if err != nil {
		return err
	}

	m.rwlock.Lock()
	defer m.rwlock.Unlock()

	mapping, ok := m.dataFrom[node]
	if !ok {
		return errors.New("node not exist")
	}

	// delete
	{
		for _, v := range ddata {
			serviceMap, ok := mapping.ServiceMapping[v.ServiceName]
			if !ok {
				continue
			}
			delete(serviceMap, v.NodeName)
			if len(serviceMap) == 0 {
				delete(mapping.ServiceMapping, v.ServiceName)
			}
		}
	}
	// add
	{
		for _, v := range adata {
			serviceMap, ok := mapping.ServiceMapping[v.ServiceName]
			if !ok {
				serviceMap = make(map[string]struct {
					Addr string
				})
				mapping.ServiceMapping[v.ServiceName] = serviceMap
			}
			serviceMap[v.NodeName] = v.Info
		}
	}

	m.buildMemData()
	return nil
}

func (m *MemStore) Abandon(node string) error {
	m.rwlock.Lock()
	defer m.rwlock.Unlock()

	_, ok := m.dataFrom[node]
	if ok {
		delete(m.dataFrom, node)
		m.buildMemData()
	}
	return nil
}

func (m *MemStore) FetchFullAndWatch(node string) (map[string][]byte, error) {
	md := new(memNodeData)
	md.ServiceMapping = m.currentServices

	m.rwlock.Lock()
	defer m.rwlock.Unlock()

	// add watch
	_, ok := m.dataTo[node]
	if !ok {
		m.dataTo[node] = new(struct {
			Add []*dataTransferNodeData
			Del []*dataTransferNodeData
		})
	}

	data, err := json.Marshal(md)
	return map[string][]byte{
		_DefaultDataKey: data,
	}, err
}

func (m *MemStore) FetchChangesForPeerNodeRequest(node string) (add, del map[string][]byte, err error) {
	m.rwlock.Lock()
	defer m.rwlock.Unlock()

	s, ok := m.dataTo[node]
	if !ok {
		return nil, nil, errors.New("node not registered")
	}

	ad, err := json.Marshal(s.Add)
	if err != nil {
		return nil, nil, errors.New("marshal failed")
	}
	dd, err := json.Marshal(s.Del)
	if err != nil {
		return nil, nil, errors.New("marshal failed")
	}
	s.Add = nil
	s.Del = nil
	return map[string][]byte{
			_DefaultDataKey: ad,
		}, map[string][]byte{
			_DefaultDataKey: dd,
		}, err
}

func (m *MemStore) Unwatch(node string) error {
	m.rwlock.Lock()
	defer m.rwlock.Unlock()

	delete(m.dataTo, node)

	return nil
}

func (m *MemStore) PutDomain(domain, resolve string, domainType shared.DomainType) {
	defer m.rwlock.Unlock()
	m.rwlock.Lock()

	m.domains[dns.Fqdn(domain)] = resolve
}

func (m *MemStore) ResolveDomain(domain string, domainType shared.DomainType) (string, error) {
	defer m.rwlock.RUnlock()
	m.rwlock.RLock()

	r, ok := m.domains[domain]
	if !ok {
		return "", shared.ErrStorageNotFound
	}
	return r, nil
}

var _ Storage = new(MemStore)

func NewMemStore() *MemStore {
	store := new(MemStore)
	store.domains = make(map[string]string)
	store.services = make(map[string]map[string]struct {
		Addr string
	})
	store.currentServices = make(map[string]map[string]struct {
		Addr string
	})
	store.dataFrom = make(map[string]*memNodeData)
	store.dataTo = make(map[string]*struct {
		Add []*dataTransferNodeData
		Del []*dataTransferNodeData
	})
	return store
}
