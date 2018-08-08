package bootstrap

import (
	"sync"
	"time"
)

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
	prevRecord         *Record

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
	// maxEvict
	maxEvict int
}

func (this *recordCacheSegment) init(slots int, maxEvict int) {
	this.recordMap = make(map[string]*Record, slots)
	this.priorityQueue = make([]*Record, slots)
	this.slots = slots
	this.count = 0
	this.maxEvict = maxEvict
}

func (this *recordCacheSegment) Get(key string) (record *Record) {
	now := now()
	this.Lock()
	var ok bool
	record, ok = this.recordMap[key]
	if ok {
		// touch record
		{
			// remove
			this.removeFromLRU(record)
			// insert first
			this.insertFirstLRU(record)
		}
	}
	// try to evict keys
	this.tryToEvict(false, now)
	this.Unlock()
	return
}

func (this *recordCacheSegment) Put(key string, record *Record) {
	now := now()
	record.key = key
	this.Lock()
	// try to delete exists
	this.internalDel(key)
	if this.count >= this.slots {
		// force evict a key
		this.tryToEvict(true, now)
	}
	// put
	{
		this.recordMap[key] = record
		this.insertFirstLRU(record)
		this.insertPriorityQueue(record)
		this.count += 1
	}
	// try to evict keys
	this.tryToEvict(false, now)
	this.Unlock()
}

func (this *recordCacheSegment) Del(key string) (record *Record) {
	now := now()
	this.Lock()
	record = this.internalDel(key)
	// try to evict keys
	this.tryToEvict(false, now)
	this.Unlock()
	return
}

func (this *recordCacheSegment) internalDel(key string) (record *Record) {
	// delete item
	var ok bool
	record, ok = this.recordMap[key]
	if ok {
		delete(this.recordMap, key)
		this.removeFromLRU(record)
		// remove from queue
		this.removeFromPriorityQueue(record)
		this.count -= 1
	}
	return
}

func (this *recordCacheSegment) tryToEvict(force bool, currentTime int64) {
	var noEvict = true
	//TODO evict from PriorityQueue
	if force && noEvict {
		//TODO evict last from LRU
	}
}

func (this *recordCacheSegment) insertPriorityQueue(record *Record) {
	record.priorityQueueIndex = -1
	//TODO
}

func (this *recordCacheSegment) removeFromPriorityQueue(record *Record) {
	//TODO
}

func (this *recordCacheSegment) insertFirstLRU(record *Record) {
	record.prevRecord = nil
	record.nextRecord = nil

	record.nextRecord = this.head
	if this.head != nil {
		this.head.prevRecord = record
	}
	this.head = record
	if this.tail == nil {
		this.tail = record
	}
}

func (this *recordCacheSegment) removeFromLRU(record *Record) {
	prevRecord := record.prevRecord
	nextRecord := record.nextRecord
	if prevRecord == nil { // head one
		this.head = nextRecord
		if nextRecord == nil {
			// last one
			this.tail = nil
		} else {
			// not last
			nextRecord.prevRecord = nil
		}
	} else { // not head
		prevRecord.nextRecord = nextRecord
		if nextRecord != nil {
			// not last
			nextRecord.prevRecord = prevRecord
		} else {
			// last one
			this.tail = prevRecord
		}
	}
}

type DnsRecordCache struct {
	mask     int
	segments map[int]*recordCacheSegment
}

func (this *DnsRecordCache) init(segments, slots int, maxEvict int) {
	// multi segments support
	var i = 1
	for ; segments > i && i > 0; i <<= 1 {
	}
	if i > 0 {
		segments = i
		this.mask = i - 1
	} else {
		segments = 1
		this.mask = 0
	}
	this.segments = make(map[int]*recordCacheSegment, segments)
	for i := 0; i < segments; i++ {
		segment := new(recordCacheSegment)
		segment.init(slots, maxEvict)
		this.segments[i] = segment
	}
}

func New(segments, slots int, maxEvict int) *DnsRecordCache {
	dns := new(DnsRecordCache)
	dns.init(segments, slots, maxEvict)
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

func now() int64 {
	return time.Now().Unix()
}
