package rpcclient

import (
	"time"
)

type RPCClient struct {
	serverurl string
	timeout   time.Duration
	verbose   bool

	client Client
}

func NewRPCClient(serverurl, user, passwd string, verbose bool) *RPCClient {
	client := NewHTTPClient(serverurl)
	client.SetAuth(user, passwd)
	return &RPCClient{serverurl: serverurl, timeout: 0, verbose: verbose, client: client}
}

func (r *RPCClient) call(srvName string, in, ret interface{}) error {
	return r.client.CallTimeout(srvName, in, ret, r.timeout)
}

func (r *RPCClient) SetTimeout(timeout time.Duration) {
	r.timeout = timeout
}

func (r *RPCClient) Url() string {
	return r.serverurl + RPCPath
}
