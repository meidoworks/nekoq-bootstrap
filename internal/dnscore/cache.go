package dnscore

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

type DnsCache interface {
	Put(req, res *dns.Msg)
	Get(req *dns.Msg) *dns.Msg
}

type DnsMemCache struct {
	rwlock sync.RWMutex
	cache  map[string]struct {
		res       *dns.Msg
		timeInSec int64
		ttl       uint32
	}
	cleanUpJobTicker *time.Ticker
}

func NewDnsMemCache() DnsCache {
	cache := &DnsMemCache{
		cache: map[string]struct {
			res       *dns.Msg
			timeInSec int64
			ttl       uint32
		}{},
		cleanUpJobTicker: time.NewTicker(1 * time.Minute),
	}
	go cache.cleanupJob()
	return cache
}

func (d *DnsMemCache) Put(req, res *dns.Msg) {
	hasTTL, ttl := checkHasValidTTL(res)
	if !hasTTL {
		return
	}
	key := cacheKey(req)

	d.rwlock.Lock()
	defer d.rwlock.Unlock()
	d.cache[key] = struct {
		res       *dns.Msg
		timeInSec int64
		ttl       uint32
	}{res: res, timeInSec: time.Now().Unix(), ttl: ttl}
}

func (d *DnsMemCache) Get(req *dns.Msg) *dns.Msg {
	key := cacheKey(req)

	d.rwlock.RLock()
	defer d.rwlock.RUnlock()
	r, ok := d.cache[key]
	if !ok {
		return nil
	}
	if time.Now().Unix()-r.timeInSec > int64(r.ttl) {
		return nil
	} else {
		return r.res.Copy().SetReply(req) //FIXME perhaps we need to update ttl in the response
	}
}

func (d *DnsMemCache) cleanupJob() {
	for {
		t, ok := <-d.cleanUpJobTicker.C
		if !ok {
			break
		}
		func() {
			now := t.Unix()
			d.rwlock.Lock()
			defer d.rwlock.Unlock()
			for k, v := range d.cache {
				if now-v.timeInSec > int64(v.ttl) {
					delete(d.cache, k)
				}
			}
		}()
	}
}

func checkHasValidTTL(m *dns.Msg) (bool, uint32) {
	var ttl uint32 = math.MaxUint32
	for _, an := range m.Answer {
		if an.Header() != nil {
			if ttl > an.Header().Ttl { // use minimum ttl as the whole ttl of the record
				ttl = an.Header().Ttl
			}
		}
	}
	if ttl == math.MaxUint32 {
		ttl = 0
	}
	return ttl > 0, ttl
}

func cacheKey(m *dns.Msg) string {
	return fmt.Sprint(strings.ToLower(dns.Fqdn(m.Question[0].Name)), "::", m.Question[0].Qtype)
}
