package dnscore

import (
	"context"
	"strings"

	"github.com/miekg/dns"
)

type RequestContext struct {
	Ctx context.Context

	traceInfos []string
}

func NewRequestContext() *RequestContext {
	return &RequestContext{
		traceInfos: make([]string, 0, 4),
	}
}

func (r *RequestContext) AddTraceInfo(info string) {
	r.traceInfos = append(r.traceInfos, info)
}

func (r *RequestContext) AddTraceInfoWithDnsAnswersIfNoError(info string, msg *dns.Msg, err error) {
	if err != nil {
		return
	}
	strbldr := new(strings.Builder)
	strbldr.WriteString(info)
	strbldr.WriteByte('[')
	for _, v := range msg.Answer {
		strbldr.WriteString(v.String())
		strbldr.WriteByte(',')
	}
	strbldr.WriteByte(']')
	r.AddTraceInfo(strbldr.String())
}

func (r *RequestContext) GetTraceInfoString() string {
	return strings.Join(r.traceInfos, "|")
}
