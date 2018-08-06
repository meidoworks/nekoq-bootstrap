package bootstrap

import "sync"

type RecordType string

const (
	Record_A     RecordType = "/A"
	Record_AAAA  RecordType = "/AAAA"
	Record_NS    RecordType = "/NS"
	Record_TXT   RecordType = "/TXT"
	Record_SRV   RecordType = "/SRV"
	Record_CNAME RecordType = "/CNAME"
	Record_MX    RecordType = "/MX"
	Record_PTR   RecordType = "/PTR"
)

type Record struct {
	ExpireTime int64
}

type recordCacheSegment struct {
	sync.RWMutex
	// get record
	recordMap map[string]*Record
	// eviction
	head *Record
	// timeout
	priorityQueue []*Record
	// slots
	slots int
	// count
	count int
}

func (this *recordCacheSegment) init(slots int) {
	this.recordMap = make(map[string]*Record, slots)
	this.priorityQueue = make([]*Record, slots)
	this.slots = slots
	this.count = 0
}

func (this *recordCacheSegment) Get(key string) (record *Record) {
	this.RLock()
	record = this.recordMap[key]
	//TODO try to evict keys
	this.RUnlock()
	return
}

func (this *recordCacheSegment) Put(key string, record *Record) {
	this.Lock()
	if this.count >= this.slots {
		//TODO force evict keys
	}
	//TODO put
	//TODO try to evict keys
	this.Unlock()
}

type DnsRecordCache struct {
	mask     int
	segments map[int]*recordCacheSegment
}

func (this *DnsRecordCache) init(segments, slots int) {
	//TODO multi segments support
	segments = 1
	this.segments = make(map[int]*recordCacheSegment, segments)
	for i := 0; i < segments; i++ {
		segment := new(recordCacheSegment)
		segment.init(slots)
		this.segments[i] = segment
	}
	this.mask = 0
}

func New(segments, slots int) *DnsRecordCache {
	dns := new(DnsRecordCache)
	dns.init(segments, slots)
	return dns
}

func (this *DnsRecordCache) Get(recordType RecordType, domain string) *Record {
	var key = domain + string(recordType)
	return this.segments[this.segment(key)].Get(key)
}

func (this *DnsRecordCache) Put(recordType RecordType, domain string, record *Record) {
	var key = domain + string(recordType)
	this.segments[this.segment(key)].Put(key, record)
}

func (this *DnsRecordCache) segment(key string) int {
	hash := 0
	for i := 0; i < len(key); i++ {
		hash = 31*hash + (int(key[i]) & 0xff)
	}
	return hash & this.mask
}
