package replication

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/redis/go-redis/v9"

	"github.com/meidoworks/nekoq-bootstrap/internal/iface"
)

const (
	CommandRole       = "REPLICATOR.ROLE"
	CommandSyncAndReg = "REPLICATOR.SYNC_REG" //FIXME have to separate sync and reg
	CommandWalShip    = "REPLICATOR.SHIP"
	CommandPromote    = "REPLICATOR.PROMOTE"
)

type Resp2SimplePrimaryStandbyHandler struct {
	ps *SimplePrimaryStandby

	SnapshotSupplier iface.SnapshotSupplier
	ApplyWalLog      iface.ApplyWalLog
}

func (r Resp2SimplePrimaryStandbyHandler) Register(reg iface.RespRegister) {
	reg.AddCommandHandler(CommandRole, r.role)
	reg.AddCommandHandler(CommandSyncAndReg, r.syncAndReg)
	reg.AddCommandHandler(CommandWalShip, r.walship)
	reg.AddCommandHandler(CommandPromote, func(args []iface.RespArg) (iface.RespResult, error) {
		if err := r.ps.Promote(); err != nil {
			log.Println("promote failed:", err)
			return iface.RespErrorResult("promote failed"), nil
		} else {
			return iface.RespStringResult("OK"), nil
		}
	})
}

func (r Resp2SimplePrimaryStandbyHandler) walship(args []iface.RespArg) (iface.RespResult, error) {
	if err := r.ApplyWalLog(args[1]); err != nil {
		log.Println("apply wal log failed:", err)
		return iface.RespErrorResult("wal ship failed"), nil
	}
	return iface.RespStringResult("OK"), nil
}

func (r Resp2SimplePrimaryStandbyHandler) role([]iface.RespArg) (iface.RespResult, error) {
	switch r.ps.Role() {
	case iface.RolePrimary:
		return iface.RespIntResult(1), nil
	case iface.RoleStandby:
		return iface.RespIntResult(2), nil
	default:
		return iface.RespErrorResult("Unknown role"), nil
	}
}

func (r Resp2SimplePrimaryStandbyHandler) syncAndReg(args []iface.RespArg) (iface.RespResult, error) {
	var seq iface.SequenceId
	if err := cbor.Unmarshal(args[2], &seq); err != nil {
		log.Println("unmarshal sequence id failed:", err)
		return iface.RespErrorResult("sequence id invalid"), nil
	}
	dat, err := r.SnapshotSupplier(seq)
	if err != nil {
		log.Println("snapshot store failed:", err)
		return iface.RespErrorResult("snapshot failed"), nil
	}
	if err := r.ps.readyNode(string(args[1])); err != nil { // Note: Here we mark node ready but node may not be able to accept wal append. Node need to handle it by itself.
		log.Println("ready node failed:", err)
		return iface.RespErrorResult("node could not be ready"), nil
	}
	return iface.RespValueResult(dat), nil
}

type Resp2Client struct {
	addr string

	client *redis.Client
}

func NewResp2Client(addr string) *Resp2Client {
	c := new(Resp2Client)
	c.addr = addr
	return c
}

func (r *Resp2Client) doConnect() (iface.ReplicatorRole, error) {
	if r.client != nil {
		if role, err := r.role(); err != nil {
			// abandon broken connection and restart a new
			r.client.Close()
			r.client = nil
		} else {
			return role, nil
		}
	}
	r.client = redis.NewClient(&redis.Options{
		Addr: r.addr,
	})
	return r.role()
}

func (r *Resp2Client) role() (iface.ReplicatorRole, error) {
	res, err := r.client.Do(context.Background(), CommandRole).Result()
	if err != nil {
		return 0, err
	}
	role := res.(int)
	switch role {
	case 1:
		return iface.RolePrimary, nil
	case 2:
		return iface.RoleStandby, nil
	default:
		return 0, errors.New("unknown role")
	}
}

func (r *Resp2Client) syncAndRegister(seq iface.SequenceId, node string) ([]byte, error) {
	dat, err := cbor.Marshal(seq)
	if err != nil {
		return nil, err
	}
	res, err := r.client.Do(context.Background(), CommandSyncAndReg, node, dat).Result()
	if err != nil {
		return nil, err
	}
	return res.([]byte), nil
}

func (r *Resp2Client) walShip(dat []byte) error {
	const retryCnt = 3
	var rerr error
	for i := 0; i < retryCnt; i++ {
		res, err := r.client.Do(context.Background(), CommandWalShip, dat).Result()
		if err != nil {
			rerr = err
			log.Println("Resp2Client wal ship failed, retry...")
			time.Sleep(500 * time.Millisecond)
			continue
		} else if v := res.(string); v != "OK" {
			return errors.New("wal ship got not OK:" + v)
		} else {
			return nil
		}
	}
	return rerr
}
