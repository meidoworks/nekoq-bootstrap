package httpserver

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/meidoworks/nekoq-bootstrap/internal/iface"
)

type HttpMethod string

var (
	MethodGet     HttpMethod = http.MethodGet
	MethodPost    HttpMethod = http.MethodPost
	MethodPut     HttpMethod = http.MethodPut
	MethodDelete  HttpMethod = http.MethodDelete
	MethodOptions HttpMethod = http.MethodOptions
	MethodHead    HttpMethod = http.MethodHead
	MethodPatch   HttpMethod = http.MethodPatch
	MethodConnect HttpMethod = http.MethodConnect
	MethodTrace   HttpMethod = http.MethodTrace
)

type HttpServer struct {
	router *httprouter.Router

	listenAddr string

	server *http.Server

	GeneralErrorHandler func(err error)
}

func NewHttpServer(addr string) *HttpServer {
	h := httprouter.New()

	return &HttpServer{
		router:     h,
		listenAddr: addr,
	}
}

func (h *HttpServer) Add(method HttpMethod, path string, handler iface.HttpHandler) {
	h.router.Handle(newHandle(method, path, handler, h.GeneralErrorHandler))
}

func (h *HttpServer) Startup() error {
	ln, err := net.Listen("tcp", h.listenAddr)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr:    h.listenAddr,
		Handler: h.router,
	}
	h.server = server
	go func() {
		if err := h.server.Serve(ln); err != nil {
			log.Println("ERROR: http server:", err)
		}
	}()
	return nil
}

func (h *HttpServer) Stop() error {
	return h.server.Shutdown(context.Background())
}

func newHandle(m HttpMethod, p string, h iface.HttpHandler, ef func(err error)) (method, path string, handle httprouter.Handle) {
	return string(m), p, func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		r, err := h(request, params)
		if err != nil {
			if ef != nil {
				ef(err)
			}
			writer.WriteHeader(http.StatusInternalServerError)
			return
		} else {
			if err := r.Render(writer); err != nil {
				if ef != nil {
					ef(err)
				}
			}
		}
	}
}
