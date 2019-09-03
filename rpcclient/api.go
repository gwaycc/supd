package rpcclient

import (
	"fmt"

	"github.com/gwaycc/supd/types"
	"github.com/gwaylib/errors"
)

type StatusReply struct {
	Success bool
}

type ProcessInfoReply struct {
	ProcessInfo *types.ProcessInfo
}

type AllProcessInfoReply struct {
	AllProcessInfo []types.ProcessInfo
}

type GetVersionArg struct {
}
type GetVersionRet struct {
	Version string
}

func (r *RPCClient) GetVersion() (*GetVersionRet, error) {
	in := &GetVersionArg{}
	ret := &GetVersionRet{}
	if err := r.call("Supervisor.GetVersion", in, ret); err != nil {
		return nil, errors.As(err)
	}
	return ret, nil
}

type GetAllProcessInfoArg struct {
}
type GetAllProcessInfoRet AllProcessInfoReply

func (r *RPCClient) GetAllProcessInfo() (*GetAllProcessInfoRet, error) {
	in := &GetAllProcessInfoArg{}
	ret := &GetAllProcessInfoRet{}
	if err := r.call("Supervisor.GetAllProcessInfo", in, ret); err != nil {
		return nil, errors.As(err)
	}
	return ret, nil
}

func getChangeName(change string) (string, error) {
	srvName := ""
	switch change {
	case "start":
		srvName = "Start"
	case "stop":
		srvName = "Stop"
	case "restart":
		srvName = "Restart"
	default:
		return "", errors.New("Incorrect required state")
	}
	return srvName, nil
}

type ChangeProcessStateArg struct {
	Name string
}

type ChangeProcessStateRet struct {
	Success bool
}

func (r *RPCClient) ChangeProcessState(change string, processName string) (*ChangeProcessStateRet, error) {
	srvName, err := getChangeName(change)
	if err != nil {
		return nil, errors.As(err)
	}

	in := &ChangeProcessStateArg{processName}
	ret := &ChangeProcessStateRet{}
	if err := r.call(fmt.Sprintf("Supervisor.%sProcess", srvName), in, ret); err != nil {
		return nil, errors.As(err)
	}
	return ret, nil
}

type ChangeAllProcessStateArg struct {
	Wait bool
}
type ChangeAllProcessStateRet AllProcessInfoReply

func (r *RPCClient) ChangeAllProcessState(change string) (*ChangeAllProcessStateRet, error) {
	srvName, err := getChangeName(change)
	if err != nil {
		return nil, errors.As(err)
	}
	in := &ChangeAllProcessStateArg{true}
	ret := &ChangeAllProcessStateRet{}
	if err := r.call(fmt.Sprintf("Supervisor.%sAllProcesses", srvName), in, ret); err != nil {
		return nil, errors.As(err)
	}
	return ret, nil
}

type ShutdownArg struct {
}
type ShutdownRet struct {
	Success bool
}

func (r *RPCClient) Shutdown() (*ShutdownRet, error) {
	in := &ShutdownArg{}
	ret := &ShutdownRet{}
	if err := r.call("Supervisor.Shutdown", in, ret); err != nil {
		return nil, errors.As(err)
	}
	return ret, nil
}

type ReloadConfigArg struct {
}
type ReloadConfigRet types.ReloadConfigResult

func (r *RPCClient) ReloadConfig() (*ReloadConfigRet, error) {
	in := &ReloadConfigArg{}
	ret := &ReloadConfigRet{}
	ret.AddedGroup = make([]string, 0)
	ret.ChangedGroup = make([]string, 0)
	ret.RemovedGroup = make([]string, 0)
	if err := r.call("Supervisor.ReloadConfig", in, ret); err != nil {
		return nil, errors.As(err)
	}
	return ret, nil
}

type SignalProcessArg struct {
	ProcName string
	Signal   string
}

type SignalProcessRet types.BooleanReply

func (r *RPCClient) SignalProcess(in *SignalProcessArg) (*SignalProcessRet, error) {
	ret := &SignalProcessRet{}
	if err := r.call("Supervisor.SignalProcess", &in, ret); err != nil {
		return nil, errors.As(err)
	}
	return ret, nil
}

type SignalAllProcessesArg struct {
	Signal string
}
type SignalAllProcessesRet AllProcessInfoReply

func (r *RPCClient) SignalAllProcesses(in *SignalAllProcessesArg) (*SignalAllProcessesRet, error) {
	ret := &SignalAllProcessesRet{}
	if err := r.call("Supervisor.SignalAllProcesses", in, ret); err != nil {
		return nil, errors.As(err)
	}
	return ret, nil
}

type GetProcessInfoArg struct {
	Name string
}
type GetProcessInfoRet ProcessInfoReply

func (r *RPCClient) GetProcessInfo(in *GetProcessInfoArg) (*GetProcessInfoRet, error) {
	ret := &GetProcessInfoRet{}
	if err := r.call("Supervisor.GetProcessInfo", in, ret); err != nil {
		return nil, errors.As(err)
	}
	return ret, nil
}
