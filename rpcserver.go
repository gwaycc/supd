package supd

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strings"

	"github.com/gwaycc/supd/rpcclient"

	log "github.com/sirupsen/logrus"
)

type RPCServer struct {
	listeners map[string]net.Listener
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
	// r.HandleHTTP(rpcclient.RPCPath, rpc.DefaultDebugPath)

	return &RPCServer{
		listeners: make(map[string]net.Listener),
		started:   false,

		supervisor: s,
		rpcServer:  r,
	}
}

// stop network listening
func (p *RPCServer) Stop() {
	log.Info("stop listening")
	for _, listener := range p.listeners {
		listener.Close()
	}
	p.started = false
}

func (p *RPCServer) StartUnixHttpServer(user string, password string, listenAddr string) {
	os.Remove(listenAddr)
	p.startHttpServer(user, password, "unix", listenAddr)
}

func (p *RPCServer) StartInetHttpServer(user string, password string, listenAddr string) {
	p.startHttpServer(user, password, "tcp", listenAddr)
}

func (p *RPCServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.rpcServer.ServeHTTP(w, r)
}

func (p *RPCServer) startHttpServer(user string, password string, protocol string, listenAddr string) {
	if p.started {
		return
	}
	p.started = true
	s := p.supervisor

	http.Handle(rpcclient.RPCPath, NewHttpBasicAuth(user, password, p))
	prog_rest_handler := NewSupervisorRestful(s).CreateProgramHandler()
	http.Handle("/program/", NewHttpBasicAuth(user, password, prog_rest_handler))
	listener, err := net.Listen(protocol, listenAddr)
	if err != nil {
		log.WithFields(log.Fields{"addr": listenAddr, "protocol": protocol}).Fatal("fail to listen on address")
		return
	}

	log.WithFields(log.Fields{"addr": listenAddr, "protocol": protocol}).Info("success to listen on address")
	p.listeners[protocol] = listener
	http.Serve(listener, nil)
}
