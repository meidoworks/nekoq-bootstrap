package dnsdyn

import (
	"context"
	"strings"
	"sync/atomic"

	"github.com/BurntSushi/toml"
	"github.com/meidoworks/nekoq-component/configure/configapi"
	"github.com/meidoworks/nekoq-component/configure/configclient"
	"github.com/miekg/dns"

	"github.com/meidoworks/nekoq-bootstrap/internal/dnscore"
	"github.com/meidoworks/nekoq-bootstrap/internal/shared"
)

type ConfigureContainer struct {
	A    map[string]string `toml:"A"`
	TXT  map[string]string `toml:"TXT"`
	SRV  map[string]string `toml:"SRV"`
	PTR  map[string]string `toml:"PTR"`
	AAAA map[string]string `toml:"AAAA"`
}

type ResolveContainer struct {
	A    map[string]string
	TXT  map[string]string
	SRV  map[string]string
	PTR  map[string]string
	AAAA map[string]string
}

func NewResolveContainer() *ResolveContainer {
	return &ResolveContainer{
		A:    map[string]string{},
		TXT:  map[string]string{},
		SRV:  map[string]string{},
		PTR:  map[string]string{},
		AAAA: map[string]string{},
	}
}

type DnsDynConfStore struct {
	serverLists []string

	client *configclient.Client
	adv    *configclient.ClientAdv[*ConfigureContainer]

	container *atomic.Value
}

func (d *DnsDynConfStore) GetContainer() *ResolveContainer {
	return d.container.Load().(*ResolveContainer)
}

func NewDnsDynConfStore(serverList []string) *DnsDynConfStore {
	rcVal := new(atomic.Value)
	rcVal.Store(NewResolveContainer())
	return &DnsDynConfStore{
		serverLists: serverList,
		container:   rcVal,
	}
}

func (d *DnsDynConfStore) Startup() error {
	sel := new(configapi.Selectors)
	if err := sel.Fill("app=nekoq-bootstrap,dc=default,env=PROD"); err != nil {
		return err
	}
	client := configclient.NewClient(d.serverLists, configclient.ClientOptions{
		OverrideSelectors: sel,
	})
	adv := configclient.NewClientAdv[*ConfigureContainer](client)
	adv.OnChange = d.onChange
	_, err := adv.Register("nekoq-bootstrap.dns", "records", toml.Unmarshal)
	if err != nil {
		defer func(client *configclient.Client) {
			err := client.StopClient()
			if err != nil {
				logger.Error("stop client failed:", err)
			}
		}(client)
		return err
	}
	d.client = client
	d.adv = adv
	if err := client.StartClient(); err != nil {
		defer func(client *configclient.Client) {
			err := client.StopClient()
			if err != nil {
				logger.Error("stop client failed:", err)
			}
		}(client)
		return err
	}
	if err := client.WaitStartupConfigureLoaded(context.Background()); err != nil {
		logger.Error("wait startupConfigureLoaded failed:", err)
		return err
	}
	return nil
}

func (d *DnsDynConfStore) Stop() error {
	return d.client.StopClient()
}

func (d *DnsDynConfStore) onChange(cfg configapi.Configuration, container *ConfigureContainer) {
	logger.Info("receive dns record change.")
	if err := d.process(container); err != nil {
		logger.Error("process dns record change failed:", err)
	} else {
		logger.Info("process dns record change success.")
	}
}

func (d *DnsDynConfStore) ResolveDomain(domain string, domainType shared.DomainType) (string, error) {
	domain = strings.ToLower(domain)

	switch domainType {
	case shared.DomainTypeA:
		val, ok := d.GetContainer().A[domain]
		if ok {
			return val, nil
		}
	case shared.DomainTypeTxt:
		val, ok := d.GetContainer().TXT[domain]
		if ok {
			return val, nil
		}
	case shared.DomainTypeSrv:
		val, ok := d.GetContainer().SRV[domain]
		if ok {
			return val, nil
		}
	case shared.DomainTypePtr:
		val, ok := d.GetContainer().PTR[domain]
		if ok {
			return val, nil
		}
	case shared.DomainTypeAAAA:
		val, ok := d.GetContainer().AAAA[domain]
		if ok {
			return val, nil
		}
	}

	return "", shared.ErrStorageNotFound
}

func (d *DnsDynConfStore) PutDomain(domain, resolve string, domainType shared.DomainType) {
	panic("unsupported")
}

func (d *DnsDynConfStore) process(container *ConfigureContainer) error {
	rc := NewResolveContainer()
	for key, val := range container.A {
		rc.A[dns.Fqdn(strings.ToLower(key))] = val
	}
	for key, val := range container.TXT {
		rc.TXT[dns.Fqdn(strings.ToLower(key))] = val
	}
	for key, val := range container.SRV {
		rc.SRV[dns.Fqdn(strings.ToLower(key))] = val
	}
	for key, val := range container.PTR {
		domain := dnscore.FromIPAddressToPtrFqdn(key)
		resolve := dns.Fqdn(strings.ToLower(val))
		rc.PTR[domain] = resolve
	}
	for key, val := range container.AAAA {
		rc.AAAA[dns.Fqdn(strings.ToLower(key))] = val
	}
	d.container.Store(rc)
	return nil
}
