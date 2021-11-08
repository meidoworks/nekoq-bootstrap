package bootstrap

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
)

type SyncReq struct {
	NodeId string
	Full   map[string][]byte

	Add map[string][]byte
	Del map[string][]byte

	Err chan error
}

type HaModule struct {
	NodeId  string
	Listen  string
	Storage Storage

	NodePeerMapping   map[string]string
	ClientPeerMapping map[string]*struct {
		LastUpdate int64
	}
	ClientPeerMappingLock sync.Mutex

	ClusterName   string
	ClusterSecret string

	SyncQueue chan SyncReq

	DebugPrint bool

	router *httprouter.Router
}

func newSyncReq(nodeId string) SyncReq {
	return SyncReq{
		NodeId: nodeId,
		Err:    make(chan error, 1),
	}
}

func NewHaModule(node, listen, clusterName, clusterSecret string, peerMapping map[string]string, storage Storage) (*HaModule, error) {
	ha := new(HaModule)
	ha.SyncQueue = make(chan SyncReq, 1024)
	ha.NodePeerMapping = make(map[string]string)
	ha.ClusterSecret = clusterSecret
	ha.ClusterName = clusterName
	ha.Storage = storage
	ha.NodeId = node
	ha.Listen = listen
	ha.ClientPeerMapping = make(map[string]*struct {
		LastUpdate int64
	})

	for k, v := range peerMapping {
		// skip self node
		if k == node {
			continue
		}
		ha.NodePeerMapping[k] = v
	}

	return ha, nil
}

func (this *HaModule) StartSync() error {

	// as client
	{
		// single thread processor for merging data
		go this.ProcessSyncWorker()

		// start peer fetching worker
		for node, peer := range this.NodePeerMapping {
			go this.PeerSyncWorker(peer, node)
		}
	}

	// as server
	{
		// check peer health
		go this.CheckPeerHealth()
	}
	return this.HttpEndpoint()
}

func (this *HaModule) CheckPeerHealth() {
	for {
		const interval = 20
		time.Sleep(1 * time.Second)
		barrier := time.Now().UnixMilli() - interval*1000

		f := func() {
			this.ClientPeerMappingLock.Lock()
			defer this.ClientPeerMappingLock.Unlock()
			var rlist []string
			for node, peer := range this.ClientPeerMapping {
				if peer.LastUpdate < barrier {
					rlist = append(rlist, node)
					log.Println("[INFO] find health check failed node. node:", node)
					err := this.Storage.Unwatch(node)
					if err != nil {
						log.Println("[ERROR] health check decision: detach node:"+node, "failed. msg:", err)
					}
				}
			}
			for _, v := range rlist {
				delete(this.ClientPeerMapping, v)
			}
		}
		f()
	}
}

func (this *HaModule) ProcessSyncWorker() {
	for {
		req := <-this.SyncQueue

		if req.Full != nil {
			if err := this.Storage.FullFrom(req.NodeId, req.Full); err != nil {
				req.Err <- err
			} else {
				req.Err <- nil
			}
			continue
		}
		if req.Add != nil || req.Del != nil {
			if err := this.Storage.SyncFrom(req.NodeId, req.NodeId, req.Add, req.Del); err != nil {
				req.Err <- err
			} else {
				req.Err <- nil
			}
			continue
		}
	}
}

func (this *HaModule) PeerSyncWorker(addr, peerNodeId string) {
	defer func() {
		log.Println("[INFO] PeerSyncWorker exist. nodeId:", peerNodeId)
	}()
MainLoop:
	for {
		time.Sleep(500 * time.Millisecond)
		log.Println("[INFO] start syncing from node:", peerNodeId)
		// cleanup previous data
		err := this.Storage.Abandon(peerNodeId)
		if err != nil {
			log.Println("[ERROR] detach node:", peerNodeId, "error. msg:", err)
			continue
		}

		// Connect to remote
		c, err := this.connectRemote(addr)
		if err != nil {
			log.Println("[ERROR] connecting:"+fmt.Sprintf("%s_%s", peerNodeId, addr), "failed. waiting for retry... msg:", err)
			continue
		}

		// query full data set for node initialization
		{
			data, err := this.queryFullDataSet(addr, c)
			if err != nil {
				log.Println("[ERROR] query full data set from node:", peerNodeId, "error. msg:", err)
				continue
			}
			fullReq := newSyncReq(peerNodeId)
			fullReq.Full = data
			this.SyncQueue <- fullReq
			if err := <-fullReq.Err; err != nil {
				log.Println("[ERROR] store full data set from node:", peerNodeId, "error. msg:", err)
				continue
			}
		}

		// query incremental
		{
			queryCount := 0
			retryCnt := 0
		NextTrigger:
			for {
				time.Sleep(1000 * time.Millisecond)

				// every N seconds to retrieve full data set
				// fetch incremental data within the period
				if queryCount < rand.Intn(20)+590 {
					// incremental fetch

					log.Println("[INFO] trigger query incremental data set for node:", peerNodeId)
					add, del, err := this.queryIncrementalData(addr, c)
					if err != nil {
						log.Println("[ERROR] store incremental data set from node:", peerNodeId, "error. msg:", err)
						retryCnt++
						if retryCnt > 10 {
							log.Println("[ERROR] reach max retry count when storing incremental. restart syncing node:", peerNodeId)
							continue MainLoop
						}
						// FIXME retry may not be appropriate
						// because queryIncrementalData has side effect which will delete incremental data at once
						continue NextTrigger
					}
					retryCnt = 0

					incReq := newSyncReq(peerNodeId)
					incReq.Add = add
					incReq.Del = del
					this.SyncQueue <- incReq
					if err := <-incReq.Err; err != nil {
						log.Println("[ERROR] store inc data set from node:", peerNodeId, "error. restart syncing. msg:", err)
						continue MainLoop
					}
					queryCount++
					continue NextTrigger
				} else {
					// full fetch

					log.Println("[INFO] trigger query full data set for node:", peerNodeId)
					data, err := this.queryFullDataSet(addr, c)
					if err != nil {
						log.Println("[ERROR] query full data set from node:", peerNodeId, "error. msg:", err)
						time.Sleep(500 * time.Millisecond)
						continue MainLoop
					}
					fullReq := newSyncReq(peerNodeId)
					fullReq.Full = data
					this.SyncQueue <- fullReq
					if err := <-fullReq.Err; err != nil {
						log.Println("[ERROR] store full data set from node:", peerNodeId, "error. msg:", err)
						continue MainLoop
					}
					queryCount = 0
					continue NextTrigger
				}
			}
		}
	}
}
