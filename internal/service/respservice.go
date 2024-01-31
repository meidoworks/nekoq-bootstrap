package service

import (
	"log"
	"strings"

	"github.com/tidwall/redcon"

	"github.com/meidoworks/nekoq-bootstrap/internal/iface"
)

type Resp2Service struct {
	server         *redcon.Server
	commandMapping map[string]iface.RespCommandHandler

	config *RespServiceConfig
}

type RespServiceConfig struct {
	Addr string
}

func NewResp2Service(config *RespServiceConfig) *Resp2Service {
	r := new(Resp2Service)

	server := redcon.NewServerNetwork("tcp", config.Addr, r.processCommand, r.acceptConn, r.closedConn)
	r.server = server
	r.config = config
	r.commandMapping = make(map[string]iface.RespCommandHandler)
	r.AddCommandHandler("PING", func(args []iface.RespArg) (iface.RespResult, error) {
		return iface.RespStringResult("PONG"), nil
	})

	return r
}

func (r *Resp2Service) ServeAndWait() error {
	return r.server.ListenAndServe()
}

func (r *Resp2Service) processCommand(conn redcon.Conn, cmd redcon.Command) {
	c := strings.ToLower(string(cmd.Args[0]))
	h, ok := r.commandMapping[c]
	if !ok {
		conn.WriteError("Unknown Command:" + c)
		return
	}
	var args []iface.RespArg
	for _, v := range cmd.Args {
		args = append(args, v)
	}
	result, err := h(args)
	if err != nil {
		log.Println("Resp2Service process command failed:", err)
		conn.WriteError("Failure")
		return
	}
	if err := result(conn); err != nil {
		log.Println("Resp2Service render result failed:", err)
		return
	}
}

func (r *Resp2Service) AddCommandHandler(command string, h iface.RespCommandHandler) {
	r.commandMapping[strings.ToLower(command)] = h
}

func (r *Resp2Service) acceptConn(conn redcon.Conn) bool {
	log.Println("Resp2Service accept redis proto peer connection:", conn.RemoteAddr())
	return true
}

func (r *Resp2Service) closedConn(conn redcon.Conn, err error) {
	log.Println("Resp2Service close redis proto peer connection:", conn, "error:", err)
}
