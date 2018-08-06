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
	key string

	priorityQueueIndex int
	nextRecord         *Record

	ExpireTime int64
}

type recordCacheSegment struct {
	sync.Mutex
	// get record
	recordMap map[string]*Record
	// eviction
	head *Record
	tail *Record
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
	this.Lock()
	record = this.recordMap[key]
	//TODO touch record
	// try to evict keys
	this.tryToEvict(false)
	this.Unlock()
	return
}

func (this *recordCacheSegment) Put(key string, record *Record) {
	record.key = key
	this.Lock()
	// try to delete exists
	this.internalDel(key)
	if this.count >= this.slots {
		// force evict a key
		this.tryToEvict(true)
	}
	//TODO put
	// try to evict keys
	this.tryToEvict(false)
	this.Unlock()
}

func (this *recordCacheSegment) Del(key string) (record *Record) {
	this.Lock()
	record = this.internalDel(key)
	// try to evict keys
	this.tryToEvict(false)
	this.Unlock()
}

func (this *recordCacheSegment) internalDel(key string) (record *Record) {
	//TODO delete item
}

func (this *recordCacheSegment) tryToEvict(force bool) {
	//TODO
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

func (this *DnsRecordCache) Del(recordType RecordType, domain string) *Record {
	var key = domain + string(recordType)
	return this.segments[this.segment(key)].Del(key)
}

func (this *DnsRecordCache) segment(key string) int {
	hash := 0
	for i := 0; i < len(key); i++ {
		hash = 31*hash + (int(key[i]) & 0xff)
	}
	return hash & this.mask
}
