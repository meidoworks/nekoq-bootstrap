package iface

import "github.com/tidwall/redcon"

type RespRegister interface {
	AddCommandHandler(command string, h RespCommandHandler)
}

type RespResult func(conn redcon.Conn) error

type RespArg []byte

type RespCommandHandler func(args []RespArg) (RespResult, error)

func RespErrorResult(msg string) RespResult {
	return func(conn redcon.Conn) error {
		conn.WriteError(msg)
		return nil
	}
}

var nilResult = func(conn redcon.Conn) error {
	conn.WriteNull()
	return nil
}

func RespNilResult() RespResult {
	return nilResult
}

func RespValueResult(val []byte) RespResult {
	return func(conn redcon.Conn) error {
		conn.WriteBulk(val)
		return nil
	}
}

func RespStringResult(msg string) RespResult {
	return func(conn redcon.Conn) error {
		conn.WriteString(msg)
		return nil
	}
}

func RespIntResult(i int) RespResult {
	return func(conn redcon.Conn) error {
		conn.WriteInt(i)
		return nil
	}
}
