package supd

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gwaycc/supd/rpcclient"
	"github.com/gwaylib/errors"

	log "github.com/sirupsen/logrus"
)

var (
	HttpMux = &HttpServeMux{}
)

type HttpServeMux struct {
	http.ServeMux
}

func (s *HttpServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// For debug
	s.ServeMux.ServeHTTP(w, r)
}

type RPCServer struct {
	listeners map[string]*http.Server
	// true if RPC is started
	started bool

	supervisor *Supervisor
	rpcServer  *rpc.Server
}

type httpBasicAuth struct {
	user     string
	password string
	handler  http.Handler
}

func NewHttpBasicAuth(user string, password string, handler http.Handler) *httpBasicAuth {

	return &httpBasicAuth{user: user, password: password, handler: handler}
}

func (h *httpBasicAuth) SetAuth(user, passwd string, handler http.Handler) {
	h.user = user
	h.password = passwd
	h.handler = handler
}

func (h *httpBasicAuth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.user == "" || h.password == "" {
		log.Debug("no auth required")
		h.handler.ServeHTTP(w, r)
		return
	}
	log.Debug("require authentication")
	username, password, ok := r.BasicAuth()
	if ok && username == h.user {
		if strings.HasPrefix(h.password, "{SHA}") {
			log.Debug("auth with SHA")
			hash := sha1.New()
			io.WriteString(hash, password)
			if hex.EncodeToString(hash.Sum(nil)) == h.password[5:] {
				h.handler.ServeHTTP(w, r)
				return
			}
		} else if password == h.password {
			log.Debug("Auth with normal password")
			h.handler.ServeHTTP(w, r)
			return
		}
	}
	w.Header().Set("WWW-Authenticate", "Basic realm=\"supervisor\"")
	w.WriteHeader(401)
}

func NewRPCServer(s *Supervisor) *RPCServer {
	r := rpc.NewServer()
	if err := r.Register(s); err != nil {
		log.Fatal(err)
	}

	return &RPCServer{
		listeners: make(map[string]*http.Server),
		started:   false,

		supervisor: s,
		rpcServer:  r,
	}
}

// stop network listening
func (p *RPCServer) Stop() {
	log.Info("stop listening")
	for key, listener := range p.listeners {
		listener.Close()
		delete(p.listeners, key)
	}
	p.started = false
}

func (p *RPCServer) StartUnixHttpServer(user string, password string, listenAddr string) {
	os.Remove(listenAddr)
	os.MkdirAll(filepath.Dir(listenAddr), 0755)
	p.startHttpServer(user, password, "unix", listenAddr)
}

func (p *RPCServer) StartInetHttpServer(user string, password string, listenAddr string) {
	p.startHttpServer(user, password, "tcp", listenAddr)
}

func (p *RPCServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.rpcServer.ServeHTTP(w, r)
}

var (
	rpcAuthHandle     *httpBasicAuth
	programAuthHandle *httpBasicAuth
)

func (p *RPCServer) startHttpServer(user string, password string, protocol string, listenAddr string) {
	if p.started {
		return
	}
	p.started = true
	s := p.supervisor

	if rpcAuthHandle == nil {
		rpcAuthHandle = NewHttpBasicAuth(user, password, p)
		HttpMux.Handle(rpcclient.RPCPath, rpcAuthHandle)
	} else {
		rpcAuthHandle.SetAuth(user, password, p)
	}

	prog_rest_handler := NewSupervisorRestful(s).CreateProgramHandler()
	if programAuthHandle == nil {
		programAuthHandle = NewHttpBasicAuth(user, password, prog_rest_handler)
		HttpMux.Handle("/program/", programAuthHandle)
	} else {
		programAuthHandle.SetAuth(user, password, prog_rest_handler)
	}

	httpServer, ok := p.listeners[protocol]
	if ok {
		if err := httpServer.Close(); err != nil {
			log.Warn(errors.As(err))
		}
		httpServer = nil
		time.Sleep(1e9)
	}
	listener, err := net.Listen(protocol, listenAddr)
	if err != nil {
		log.Warn(errors.As(err))
		return
	}
	log.WithFields(log.Fields{"addr": listenAddr, "protocol": protocol}).Info("success to listen on address")
	httpServer = &http.Server{Handler: HttpMux}
	p.listeners[protocol] = httpServer
	if err := httpServer.Serve(listener); err != nil && !errors.Equal(http.ErrServerClosed, err) {
		log.Warn(errors.As(err))
	}
}
