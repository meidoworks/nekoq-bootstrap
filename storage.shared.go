package bootstrap

import "errors"

var ErrStorageNotFound = errors.New("not found")

type DomainType int

const (
	DomainTypeA = 1
)

type ServiceItem struct {
	Addr   string `json:"address"`
	NodeId string `json:"node_id"`
}

type Storage interface {
	ResolveDomain(domain string, domainType DomainType) (string, error)
	PutDomain(domain, resolve string, domainType DomainType)

	GetServiceList(service string) ([]*ServiceItem, error)
	PublishService(service string, item *ServiceItem) error
	DeleteService(service string, item *ServiceItem) error

	/*
		for High Availability - client(listener) side
	*/
	FullFrom(node string, data map[string][]byte) error // get and watch
	SyncFrom(node, origNode string, add, del map[string][]byte) error
	Abandon(node string) error
	/*
		for High Availability - server(source) side
	*/
	FetchFullAndWatch(node string) (map[string][]byte, error)
	FetchChangesForPeerNodeRequest(node string) (add, del map[string][]byte, err error)
	Unwatch(node string) error
}
