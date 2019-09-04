package supd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gwaycc/supd/config"
	"github.com/gwaycc/supd/rpcclient"
	"github.com/gwaycc/supd/types"

	"github.com/gwaylib/errors"
	"github.com/hpcloud/tail"
)

func CtlOutput(data interface{}) {
	if ctlCommand.Encode == "json" {
		json.NewEncoder(os.Stdout).Encode(data)
		return
	}
	fmt.Println(data)
}

type CtlCommand struct {
	ServerUrl string `short:"s" long:"serverurl" description:"URL on which supervisord server is listening"`
	User      string `short:"u" long:"user" description:"the user name"`
	Password  string `short:"P" long:"password" description:"the password"`
	Verbose   bool   `short:"v" long:"verbose" description:"Show verbose debug information"`
	Encode    string `short:"o" long:"output" description:"set output encode for ctl command, value is txt or json, default is txt"`

	Follow bool `short:"f" long:"follow" description:"The -f option causes tail to not stop when end of file is reached."`
}

type StatusCommand struct {
}

type StartCommand struct {
}

type StopCommand struct {
}

type RestartCommand struct {
}

type ShutdownCommand struct {
}

type ReloadCommand struct {
}

type PidCommand struct {
}

type SignalCommand struct {
}

type TailCommand struct {
	Follow bool `short:"f" description:"The -f option causes tail to not stop when end of file is reached."`
}

var ctlCommand CtlCommand
var statusCommand StatusCommand
var startCommand StartCommand
var stopCommand StopCommand
var restartCommand RestartCommand
var shutdownCommand ShutdownCommand
var reloadCommand ReloadCommand
var pidCommand PidCommand
var signalCommand SignalCommand
var tailCommand TailCommand

func (x *CtlCommand) getServerUrl() string {
	options.Configuration, _ = findSupervisordConf()

	if x.ServerUrl != "" {
		return x.ServerUrl
	} else if _, err := os.Stat(options.Configuration); err == nil {
		config := config.NewConfig(options.Configuration)
		config.Load()
		if entry, ok := config.GetSupervisorctl(); ok {
			serverurl := entry.GetString("serverurl", "")
			if serverurl != "" {
				return serverurl
			}
		}
	}
	return "http://localhost:9002"
}

func (x *CtlCommand) getUser() string {
	options.Configuration, _ = findSupervisordConf()

	if x.User != "" {
		return x.User
	} else if _, err := os.Stat(options.Configuration); err == nil {
		config := config.NewConfig(options.Configuration)
		config.Load()
		if entry, ok := config.GetSupervisorctl(); ok {
			user := entry.GetString("username", "")
			return user
		}
	}
	return ""
}

func (x *CtlCommand) getPassword() string {
	options.Configuration, _ = findSupervisordConf()

	if x.Password != "" {
		return x.Password
	} else if _, err := os.Stat(options.Configuration); err == nil {
		config := config.NewConfig(options.Configuration)
		config.Load()
		if entry, ok := config.GetSupervisorctl(); ok {
			password := entry.GetString("password", "")
			return password
		}
	}
	return ""
}

func (x *CtlCommand) createRpcClient() *rpcclient.RPCClient {
	rpcc := rpcclient.NewRPCClient(x.getServerUrl(), x.getUser(), x.getPassword(), x.Verbose)
	return rpcc
}

func (x *CtlCommand) Execute(args []string) error {
	if len(args) == 0 {
		return nil
	}

	rpcc := x.createRpcClient()
	verb := args[0]

	switch verb {

	////////////////////////////////////////////////////////////////////////////////
	// STATUS
	////////////////////////////////////////////////////////////////////////////////
	case "status":
		x.status(rpcc, args[1:])

		////////////////////////////////////////////////////////////////////////////////
		// START or STOP
		////////////////////////////////////////////////////////////////////////////////
	case "start", "stop", "restart":
		x.startStopProcesses(rpcc, verb, args[1:])

		////////////////////////////////////////////////////////////////////////////////
		// SHUTDOWN
		////////////////////////////////////////////////////////////////////////////////
	case "shutdown":
		x.shutdown(rpcc)
	case "reload":
		x.reload(rpcc)
	case "signal":
		sig_name, processes := args[1], args[2:]
		x.signal(rpcc, sig_name, processes)
	case "pid":
		x.getPid(rpcc, args[1])
	case "tail":
		return tailCommand.Execute(args[1:])
	default:
		CtlOutput("unknown command")
	}

	return nil
}

// get the status of processes
func (x *CtlCommand) status(rpcc *rpcclient.RPCClient, processes []string) {
	processesMap := make(map[string]bool)
	for _, process := range processes {
		processesMap[process] = true
	}
	ret, err := rpcc.GetAllProcessInfo()
	if err != nil {
		CtlOutput(errors.As(err))
		os.Exit(1)
		return
	}
	x.showProcessInfo(ret.AllProcessInfo, processesMap)
}

// start or stop the processes
// verb must be: start or stop
func (x *CtlCommand) startStopProcesses(rpcc *rpcclient.RPCClient, verb string, processes []string) {
	state := map[string]string{
		"start":   "started",
		"stop":    "stopped",
		"restart": "restarted",
	}
	x._startStopProcesses(rpcc, verb, processes, state[verb], true)
}

func (x *CtlCommand) _startStopProcesses(rpcc *rpcclient.RPCClient, verb string, processes []string, state string, showProcessInfo bool) {
	if len(processes) <= 0 {
		fmt.Printf("Please specify process for %s\n", verb)
	}
	for _, pname := range processes {
		if pname == "all" {
			reply, err := rpcc.ChangeAllProcessState(verb)
			if err == nil {
				if showProcessInfo {
					x.showProcessInfo(reply.AllProcessInfo, make(map[string]bool))
				}
			} else {
				fmt.Printf("Fail to change all process state to %s", state)
			}
		} else {
			if reply, err := rpcc.ChangeProcessState(verb, pname); err == nil {
				if showProcessInfo {
					fmt.Printf("%s: ", pname)
					if !reply.Success {
						fmt.Printf("not ")
					}
					fmt.Printf("%s\n", state)
				}
			} else {
				fmt.Printf("%s: failed [%v]\n", pname, err)
				os.Exit(1)
			}
		}
	}
}

func (x *CtlCommand) restartProcesses(rpcc *rpcclient.RPCClient, processes []string) {
	x._startStopProcesses(rpcc, "restart", processes, "restarted", true)
}

// shutdown the supervisord
func (x *CtlCommand) shutdown(rpcc *rpcclient.RPCClient) {
	if reply, err := rpcc.Shutdown(); err == nil {
		if reply.Success {
			fmt.Printf("Shut Down\n")
		} else {
			fmt.Printf("Hmmm! Something gone wrong?!\n")
		}
	} else {
		os.Exit(1)
	}
}

// reload all the programs in the supervisord
func (x *CtlCommand) reload(rpcc *rpcclient.RPCClient) {
	if reply, err := rpcc.ReloadConfig(); err == nil {

		if len(reply.AddedGroup) > 0 {
			fmt.Printf("Added Groups: %s\n", strings.Join(reply.AddedGroup, ","))
		}
		if len(reply.ChangedGroup) > 0 {
			fmt.Printf("Changed Groups: %s\n", strings.Join(reply.ChangedGroup, ","))
		}
		if len(reply.RemovedGroup) > 0 {
			fmt.Printf("Removed Groups: %s\n", strings.Join(reply.RemovedGroup, ","))
		}
	} else {
		os.Exit(1)
	}
}

// send signal to one or more processes
func (x *CtlCommand) signal(rpcc *rpcclient.RPCClient, sig_name string, processes []string) {
	for _, process := range processes {
		if process == "all" {
			reply, err := rpcc.SignalAllProcesses(&rpcclient.SignalAllProcessesArg{
				Signal: sig_name,
			})
			if err == nil {
				x.showProcessInfo(reply.AllProcessInfo, make(map[string]bool))
			} else {
				fmt.Printf("Fail to send signal %s to all process", sig_name)
				os.Exit(1)
			}
		} else {
			reply, err := rpcc.SignalProcess(&rpcclient.SignalProcessArg{
				ProcName: process,
				Signal:   sig_name,
			})
			if err == nil && reply.Success {
				fmt.Printf("Succeed to send signal %s to process %s\n", sig_name, process)
			} else {
				fmt.Printf("Fail to send signal %s to process %s\n", sig_name, process)
				os.Exit(1)
			}
		}
	}
}

// get the pid of running program
func (x *CtlCommand) getPid(rpcc *rpcclient.RPCClient, process string) {
	ret, err := rpcc.GetProcessInfo(&rpcclient.GetProcessInfoArg{process})
	if err != nil {
		fmt.Printf("program '%s' not found\n", process)
		os.Exit(1)
		return
	}
	fmt.Printf("%d\n", ret.ProcessInfo.Pid)
}

// check if group name should be displayed
func (x *CtlCommand) showGroupName() bool {
	val, ok := os.LookupEnv("SUPERVISOR_GROUP_DISPLAY")
	if !ok {
		return false
	}

	val = strings.ToLower(val)
	return val == "yes" || val == "true" || val == "y" || val == "t" || val == "1"
}

func (x *CtlCommand) showProcessInfo(allInfo []types.ProcessInfo, processesMap map[string]bool) {
	if x.Encode == "json" {
		CtlOutput(allInfo)
		return
	}
	for _, pinfo := range allInfo {
		description := pinfo.Description
		if x.inProcessMap(&pinfo, processesMap) {
			processName := pinfo.GetFullName()
			if !x.showGroupName() {
				processName = pinfo.Name
			}
			CtlOutput(fmt.Sprintf("%s%-33s %-10s%s%s", x.getANSIColor(pinfo.Statename), processName, pinfo.Statename, description, "\x1b[0m"))
		}
	}
}

func (x *CtlCommand) inProcessMap(procInfo *types.ProcessInfo, processesMap map[string]bool) bool {
	if len(processesMap) <= 0 {
		return true
	}
	for procName, _ := range processesMap {
		if procName == procInfo.Name || procName == procInfo.GetFullName() {
			return true
		}

		// check the wildcast '*'
		pos := strings.Index(procName, ":")
		if pos != -1 {
			groupName := procName[0:pos]
			programName := procName[pos+1:]
			if programName == "*" && groupName == procInfo.Group {
				return true
			}
		}
	}
	return false
}

func (x *CtlCommand) getANSIColor(statename string) string {
	if statename == "RUNNING" {
		// green
		return "\x1b[0;32m"
	} else if statename == "BACKOFF" || statename == "FATAL" {
		// red
		return "\x1b[0;31m"
	} else {
		// yellow
		return "\x1b[1;33m"
	}
}

func (c *StatusCommand) Execute(args []string) error {
	ctlCommand.status(ctlCommand.createRpcClient(), args)
	return nil
}

func (c *StartCommand) Execute(args []string) error {
	ctlCommand.startStopProcesses(ctlCommand.createRpcClient(), "start", args)
	return nil
}

func (c *StopCommand) Execute(args []string) error {
	ctlCommand.startStopProcesses(ctlCommand.createRpcClient(), "stop", args)
	return nil
}

func (rc *RestartCommand) Execute(args []string) error {
	ctlCommand.restartProcesses(ctlCommand.createRpcClient(), args)
	return nil
}

func (c *ShutdownCommand) Execute(args []string) error {
	ctlCommand.shutdown(ctlCommand.createRpcClient())
	return nil
}

func (c *ReloadCommand) Execute(args []string) error {
	ctlCommand.reload(ctlCommand.createRpcClient())
	return nil
}

func (c *SignalCommand) Execute(args []string) error {
	if len(args) == 0 {
		CtlOutput("Need sig name and process names")
		return nil
	}
	sig_name, processes := args[0], args[1:]
	ctlCommand.signal(ctlCommand.createRpcClient(), sig_name, processes)
	return nil
}

func (c *PidCommand) Execute(args []string) error {
	if len(args) == 0 {
		CtlOutput("Need process name")
		return nil
	}
	ctlCommand.getPid(ctlCommand.createRpcClient(), args[0])
	return nil
}
func (c *TailCommand) Execute(args []string) error {
	if len(args) == 0 {
		CtlOutput("Need process name")
		return nil
	}
	process := args[0]
	std := "stdout"
	if len(args) > 1 {
		std = args[1]
	}
	rpcc := ctlCommand.createRpcClient()
	ret, err := rpcc.GetProcessInfo(&rpcclient.GetProcessInfoArg{process})
	if err != nil {
		fmt.Printf("program '%s' not found\n", process)
		os.Exit(1)
		return nil
	}
	procInfo := ret.ProcessInfo
	fileName := procInfo.StdoutLogfile
	switch std {
	case "stderr":
		fileName = procInfo.StderrLogfile
	}

	fileStat, err := os.Stat(fileName)
	if err != nil {
		CtlOutput(errors.As(err))
		os.Exit(0)
		return nil
	}
	size := fileStat.Size()
	fmt.Printf("tail : %s, size:%d\n", fileName, size)
	offset := int64(1024)
	if size < 1024 {
		offset = size
	}
	t, err := tail.TailFile(fileName, tail.Config{
		Location: &tail.SeekInfo{
			Offset: -offset,
			Whence: os.SEEK_END,
		},
		Logger: tail.DiscardingLogger,
		Follow: c.Follow,
	})
	for line := range t.Lines {
		CtlOutput(line.Text)
	}
	return nil
}

func init() {
	ctlCmd, _ := parser.AddCommand("ctl",
		"Control a running daemon",
		"The ctl subcommand resembles supervisorctl command of original daemon.",
		&ctlCommand)
	ctlCmd.AddCommand("status",
		"show program status",
		"show all or some program status",
		&statusCommand)
	ctlCmd.AddCommand("start",
		"start programs",
		"start one or more programs",
		&startCommand)
	ctlCmd.AddCommand("stop",
		"stop programs",
		"stop one or more programs",
		&stopCommand)
	ctlCmd.AddCommand("restart",
		"restart programs",
		"restart one or more programs",
		&restartCommand)
	ctlCmd.AddCommand("shutdown",
		"shutdown supervisord",
		"shutdown supervisord",
		&shutdownCommand)
	ctlCmd.AddCommand("reload",
		"reload the programs",
		"reload the programs",
		&reloadCommand)
	ctlCmd.AddCommand("signal",
		"send signal to program",
		"send signal to program",
		&signalCommand)
	ctlCmd.AddCommand("pid",
		"get the pid of specified program",
		"get the pid of specified program",
		&pidCommand)
	ctlCmd.AddCommand("tail",
		"get the log of specified program",
		"get the log of specified program",
		&tailCommand)

}
