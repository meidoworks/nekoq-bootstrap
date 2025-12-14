package dnscore

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/miekg/dns"
)

const (
	USER_AGENT = "DNS-over-HTTPS/1.0 NekoQ-Bootstrap"
)

type DnsHttp struct {
	Router *httprouter.Router

	Addr string

	endpoint             *DnsEndpoint
	DebugPrintDnsRequest bool
}

func NewHttpDns(addr string, endpoint *DnsEndpoint, debug bool) (*DnsHttp, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	r := new(DnsHttp)
	r.Addr = u.Host

	router := httprouter.New()
	router.GET("/dns-query", r.dnsQuery)
	router.POST("/dns-query", r.dnsQuery)
	r.Router = router
	r.DebugPrintDnsRequest = debug
	r.endpoint = endpoint

	return r, nil
}

func (this *DnsHttp) StartSync() error {
	return http.ListenAndServe(this.Addr, this.Router)
}

func (this *DnsHttp) dnsQuery(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	reqCtx := NewRequestContext()

	defer func() {
		err := recover()
		if err != nil {
			logger.Error("process dns-http request failed. information:", err)
			reqCtx.AddTraceInfo("error occurs:" + fmt.Sprint(err))
		}
		if this.DebugPrintDnsRequest {
			logger.Debug("Domain resolve info:", reqCtx.GetTraceInfoString())
		}
	}()

	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS, POST")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Max-Age", "3600")
	w.Header().Set("Server", USER_AGENT)
	w.Header().Set("X-Powered-By", USER_AGENT)

	if r.Method == "OPTIONS" {
		w.Header().Set("Content-Length", "0")
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/dns-message" {
		logger.Error("unsupported content-type:", contentType)
		w.WriteHeader(415)
		return
	}
	var responseType string
	for _, responseCandidate := range strings.Split(r.Header.Get("Accept"), ",") {
		responseCandidate = strings.SplitN(responseCandidate, ";", 2)[0]
		if responseCandidate == "application/dns-message" {
			responseType = "application/dns-message"
			break
		}
	}
	if responseType == "" {
		if contentType == "application/dns-message" {
			responseType = "application/dns-message"
		}
	}
	if responseType == "" {
		panic("Unknown response Content-Type")
	}

	if r.Form == nil {
		const maxMemory = 1024 * 1024
		_ = r.ParseMultipartForm(maxMemory)
	}

	var reqBin []byte
	switch r.Method {
	case http.MethodGet:
		requestBase64 := r.FormValue("dns")
		if this.DebugPrintDnsRequest {
			logger.Debug("dns-http get dns base64:", requestBase64)
		}
		requestBinary, err := base64.RawURLEncoding.DecodeString(requestBase64)
		if err != nil {
			logger.Error("decode dns raw error:", err)
			w.WriteHeader(400)
			return
		}
		reqBin = requestBinary
	case http.MethodPost:
		requestBinary, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("read dns raw error:", err)
			w.WriteHeader(400)
			return
		}
		reqBin = requestBinary
	default:
		panic(errors.New("unsupported http method"))
	}
	if len(reqBin) == 0 {
		logger.Error("dns raw is empty")
		w.WriteHeader(400)
		return
	}

	msg := new(dns.Msg)
	err := msg.Unpack(reqBin)
	if err != nil {
		logger.Error("dns unpack err:", err)
		w.WriteHeader(400)
		return
	}

	reply := this.endpoint.ProcessDnsMsg(msg, reqCtx)

	w.Header().Set("Content-Type", "application/dns-message")
	now := time.Now().UTC().Format(http.TimeFormat)
	w.Header().Set("Date", now)
	w.Header().Set("Last-Modified", now)
	w.Header().Set("Vary", "Accept")

	//FIXME If the nil handling meets the standard?
	if reply == nil {
		w.WriteHeader(404)
		return
	}

	replyBin, err := reply.Pack()
	if err != nil {
		logger.Error("dns pack err:", err)
		w.WriteHeader(500)
		return
	}
	_, err = w.Write(replyBin)
	if err != nil {
		logger.ErrorF("failed to write to client: %v\n", err)
	}

}
