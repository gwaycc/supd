package rpcclient

import "context"

type RPCClient struct {
	serverurl string
	verbose   bool

	client Client
	ctx    context.Context
}

func NewRPCClient(serverurl, user, passwd string, verbose bool) *RPCClient {
	client := NewHTTPClient(serverurl)
	client.SetAuth(user, passwd)
	return &RPCClient{serverurl: serverurl, verbose: verbose, client: client, ctx: context.TODO()}
}

func (r *RPCClient) call(srvName string, in, ret interface{}) error {
	return r.client.Call(r.ctx, srvName, in, ret)
}

func (r *RPCClient) Url() string {
	return r.serverurl + RPCPath
}
