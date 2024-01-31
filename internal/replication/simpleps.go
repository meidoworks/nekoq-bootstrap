package replication

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/meidoworks/nekoq-bootstrap/internal/iface"
)

var ErrNotPrimaryNode = errors.New("not primary node")

var _ iface.PrimaryStandbyReplicator = new(SimplePrimaryStandby)

// SimplePrimaryStandby
// This implements a simple primary-standby replication mechanism
type SimplePrimaryStandby struct {
	reg     iface.RespRegister
	handler Resp2SimplePrimaryStandbyHandler
	role    iface.ReplicatorRole

	cluster struct {
		peers map[string]*Resp2Client
		size  int
	}
	standbyData struct {
		readyNodeCount int
		peers          map[string]*Resp2Client
	}

	config     *SimplePrimaryStandbyConfig
	closeChan  chan struct{}
	sync.Mutex // wal sync control lock
}

type SimplePrimaryStandbyConfig struct {
	NodeName       string
	ClusterNodeMap map[string]struct {
		Addr string
	}

	RegHandler iface.RespRegister

	SequenceIdSupplier                  iface.SequenceIdSupplier
	SnapshotSupplier                    iface.SnapshotSupplier
	ApplySnapshotAndIncrementalFunction iface.ApplySnapshotAndIncrementalFunction
	ApplyWalLog                         iface.ApplyWalLog
}

func NewSimplePrimaryStandby(config *SimplePrimaryStandbyConfig) *SimplePrimaryStandby {
	ps := new(SimplePrimaryStandby)
	ps.config = config
	ps.reg = config.RegHandler
	ps.role = iface.RoleStandby
	ps.handler = Resp2SimplePrimaryStandbyHandler{
		ps:               ps,
		SnapshotSupplier: config.SnapshotSupplier,
		ApplyWalLog:      config.ApplyWalLog,
	}
	ps.closeChan = make(chan struct{})
	return ps
}

func (s *SimplePrimaryStandby) Role() iface.ReplicatorRole {
	return s.role
}

func (s *SimplePrimaryStandby) Writable() bool {
	return s.role == iface.RolePrimary && s.standbyData.readyNodeCount == s.cluster.size
}

func (s *SimplePrimaryStandby) Initialize() error {
	s.handler.Register(s.reg)
	for k, v := range s.config.ClusterNodeMap {
		if k == s.config.NodeName {
			continue
		}
		s.cluster.size++
		s.cluster.peers[k] = NewResp2Client(v.Addr)
		go s.peerWorker(k)
	}
	return nil
}

func (s *SimplePrimaryStandby) peerWorker(node string) {
	ready := false
MainLoop:
	for {
		select {
		case <-s.closeChan:
			break MainLoop
		default:
			connected, connPrimary := s.peerAlive(node)
			if connected {
				if !connPrimary {
					break // connect to a standby node, break 'select' block and restart a new loop cycle
				}
				if !ready {
					// do sync and register
					if err := s.doSyncAndRegister(node); err != nil {
						log.Println("SimplePrimaryStandby sync and register to peer:", node, "failed:", err)
						break // break 'select' block and restart a new loop cycle
					}
					ready = true
				}
			} else {
				ready = false
			}
		}
		time.Sleep(1 * time.Second) // next keep-alive tick
	}
}

func (s *SimplePrimaryStandby) peerAlive(node string) (connected bool, connectPrimary bool) {
	c := s.cluster.peers[node]
	if r, err := c.doConnect(); err != nil {
		log.Println("SimplePrimaryStandby connect peer:", node, "failed:", err)
		return false, false
	} else {
		return true, r == iface.RolePrimary
	}
}

func (s *SimplePrimaryStandby) doSyncAndRegister(node string) error { // standby side
	c := s.cluster.peers[node]
	curId, err := s.config.SequenceIdSupplier()
	if err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	dat, err := c.syncAndRegister(curId, s.config.NodeName)
	if err != nil {
		return err
	}
	if err := s.config.ApplySnapshotAndIncrementalFunction(dat); err != nil {
		return err
	} else {
		return nil
	}
}

func (s *SimplePrimaryStandby) readyNode(node string) error { // primary side
	s.Lock()
	defer s.Unlock()

	_, ok := s.standbyData.peers[node]
	if !ok {
		n, ok := s.config.ClusterNodeMap[node]
		if !ok {
			return errors.New("no node found:" + node)
		}
		s.standbyData.peers[node] = NewResp2Client(n.Addr)
		s.standbyData.readyNodeCount++
	}
	return nil
}

func (s *SimplePrimaryStandby) Promote() error { // primary side
	s.role = iface.RolePrimary
	return nil
}

func (s *SimplePrimaryStandby) Ship(bytes []byte) error { // primary side
	for node, v := range s.standbyData.peers {
		if err := v.walShip(bytes); err != nil {
			log.Println("SimplePrimaryStandby ship wal to node:", node, "failed:", err)
			// force invalid node
			{
				s.standbyData.readyNodeCount--
				delete(s.standbyData.peers, node)
			}
			return err
		}
	}
	return nil
}
