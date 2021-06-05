package rpcclient

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"net/url"
	"sync"
	"time"

	"github.com/gwaylib/errors"
)

const (
	RPCPath = "/RPC2"
)

type Client interface {
	Close() error
	SetAuth(username, passwd string)
	Call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error
}

type rpcClient struct {
	protocol string
	url      *url.URL

	user   string
	passwd string

	tlsCfg *tls.Config

	mux       sync.Mutex
	connected bool
	client    *rpc.Client
}

func NewClient(address string) Client {
	u, err := url.Parse(address)
	if err != nil {
		panic(err)
	}
	return &rpcClient{
		url: u,
	}
}

func NewHTTPClient(address string) Client {
	u, err := url.Parse(address)
	if err != nil {
		panic(errors.As(err, address))
	}
	return &rpcClient{
		protocol: "http",
		url:      u,
	}
}

// protocol support http and tcp
func NewTlsClient(protocol, address string, config *tls.Config) Client {
	u, err := url.Parse(address)
	if err != nil {
		panic(err)
	}
	return &rpcClient{
		protocol: protocol,
		url:      u,
		tlsCfg:   config,
	}
}

// current support basic auth for http only.
func (rc *rpcClient) SetAuth(user, passwd string) {
	rc.user = user
	rc.passwd = passwd
}

func (rc *rpcClient) Close() error {
	rc.disconn()
	return nil
}

var connected = "200 Connected to Go RPC"

func (rc *rpcClient) conn() (*rpc.Client, error) {
	rc.mux.Lock()
	defer rc.mux.Unlock()

	if rc.connected {
		return rc.client, nil
	}
	var conn net.Conn
	var err error
	switch rc.url.Scheme {
	case "unix":
		conn, err = net.DialTimeout("unix", rc.url.Host+rc.url.Path, 10*time.Second)
		if err != nil {
			return nil, errors.As(err, fmt.Sprintf("%+v", rc.url))
		}
	default:
		conn, err = net.DialTimeout("tcp", rc.url.Host, 10*time.Second)
		if err != nil {
			return nil, errors.As(err, fmt.Sprintf("%+v", rc.url))
		}
	}
	if rc.tlsCfg != nil {
		conn = tls.Client(conn, rc.tlsCfg)
	}

	if rc.protocol == "http" {
		// Require successful HTTP response
		// before switching to RPC protocol.
		req := &http.Request{
			Method: "CONNECT",
			Header: make(http.Header),
		}
		if len(rc.passwd) > 0 {
			req.SetBasicAuth(rc.user, rc.passwd)
			io.WriteString(conn, "CONNECT "+RPCPath+" HTTP/1.0\n"+"Authorization:"+req.Header.Get("Authorization")+"\n\n")
		} else {
			io.WriteString(conn, "CONNECT "+RPCPath+" HTTP/1.0\n\n")
		}
		resp, err := http.ReadResponse(bufio.NewReader(conn), req)
		if err != nil {
			conn.Close()
			return nil, &net.OpError{
				Op:   "dial-http",
				Net:  rc.url.Scheme,
				Addr: nil,
				Err:  err,
			}
		}
		if resp.Status != connected {
			conn.Close()
			return nil, errors.New("unexpected HTTP response: " + resp.Status)
		}
		// pass
	}

	rc.client = rpc.NewClient(conn)

	rc.connected = true

	return rc.client, nil
}

func (rc *rpcClient) disconn() {
	rc.mux.Lock()
	defer rc.mux.Unlock()

	if rc.connected {
		rc.connected = false
		go func(c *rpc.Client) { // blocked?
			if err := c.Close(); err != nil {
				log.Println("disconn: c.Close", err)
			}
		}(rc.client)
	}
}

var (
	ErrCallCanceled = errors.New("ctx canceled")
)

func (rc *rpcClient) call(ctx context.Context, method string, args interface{}, reply interface{}) error {
	client, err := rc.conn()
	if err != nil {
		return err
	}

	select {
	case call := <-client.Go(method, args, reply, make(chan *rpc.Call, 1)).Done:
		return errors.As(call.Error)
	case <-ctx.Done():
		return ErrCallCanceled
	}
	return nil
}

const (
	ReConnSleepTime = 2 * time.Second
)

func (rc *rpcClient) Call(ctx context.Context, method string, args interface{}, reply interface{}) error {
	if err := rc.call(ctx, method, args, reply); err != nil {
		if _, ok := err.(rpc.ServerError); ok {
			// retry if the error is by a connection.
			rc.disconn()
			time.Sleep(ReConnSleepTime)
			return errors.As(rc.call(ctx, method, args, reply))
		}
		return errors.As(err)
	}
	return nil
}
