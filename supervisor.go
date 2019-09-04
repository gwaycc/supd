package supd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gwaycc/supd/config"
	"github.com/gwaycc/supd/events"
	"github.com/gwaycc/supd/faults"
	"github.com/gwaycc/supd/logger"
	"github.com/gwaycc/supd/process"
	"github.com/gwaycc/supd/rpcclient"
	"github.com/gwaycc/supd/signals"
	"github.com/gwaycc/supd/types"
	"github.com/gwaycc/supd/util"

	"github.com/gwaylib/errors"
	log "github.com/sirupsen/logrus"
)

const (
	SUPERVISOR_VERSION = "3.0"
)

type Supervisor struct {
	config     *config.Config
	procMgr    *process.ProcessManager
	rpcServer  *RPCServer
	logger     logger.Logger
	restarting bool
}

type StartProcessArgs struct {
	Name string
	Wait bool `default:"true"`
}

type ProcessStdin struct {
	Name  string
	Chars string
}

type RemoteCommEvent struct {
	Type string
	Data string
}

type StateInfo struct {
	Statecode int    `xml:"statecode"`
	Statename string `xml:"statename"`
}

type LogReadInfo struct {
	Offset int
	Length int
}

type ProcessLogReadInfo struct {
	Name   string
	Offset int
	Length int
}

type ProcessTailLog struct {
	LogData  string
	Offset   int64
	Overflow bool
}

func NewSupervisor(configFile string) *Supervisor {
	s := &Supervisor{
		config:     config.NewConfig(configFile),
		procMgr:    process.NewProcessManager(),
		restarting: false,
	}
	s.rpcServer = NewRPCServer(s)
	return s
}

// func (s *Supervisor) GetConfig() *config.Config {
// 	return s.config
// }

func (s *Supervisor) GetVersion(args *rpcclient.GetVersionArg, reply *rpcclient.GetVersionRet) error {
	reply.Version = SUPERVISOR_VERSION
	return nil
}

func (s *Supervisor) GetSupervisorVersion(args *rpcclient.GetVersionArg, reply *rpcclient.GetVersionRet) error {
	reply.Version = SUPERVISOR_VERSION
	return nil
}

// func (s *Supervisor) GetIdentification(args *struct{}, reply *rpcclient.DataReply) error {
// 	reply.Data = s.getSupervisorId()
// 	return nil
// }

func (s *Supervisor) getSupervisorId() string {
	entry, ok := s.config.GetSupervisord()
	if !ok {
		return "supervisor"
	}
	return entry.GetString("identifier", "supervisor")
}

func (s *Supervisor) GetState(args *struct{}, reply *struct{ StateInfo StateInfo }) error {
	//statecode     statename
	//=======================
	// 2            FATAL
	// 1            RUNNING
	// 0            RESTARTING
	// -1           SHUTDOWN
	log.Debug("Get state")
	reply.StateInfo.Statecode = 1
	reply.StateInfo.Statename = "RUNNING"
	return nil
}

// Get all the name of prorams
//
// Return the name of all the programs
// func (s *Supervisor) GetPrograms() []string {
// 	return s.config.GetProgramNames()
// }
func (s *Supervisor) GetPID(args *struct{}, reply *struct{ Pid int }) error {
	reply.Pid = os.Getpid()
	return nil
}

func (s *Supervisor) ReadLog(args *LogReadInfo, reply *struct{ Log string }) error {
	data, err := s.logger.ReadLog(int64(args.Offset), int64(args.Length))
	reply.Log = data
	return err
}

func (s *Supervisor) ClearLog(args *struct{}, reply *rpcclient.StatusReply) error {
	err := s.logger.ClearAllLogFile()
	reply.Success = err == nil
	return err
}

func (s *Supervisor) Shutdown(args *struct{}, reply *rpcclient.StatusReply) error {
	reply.Success = true
	log.Info("received rpc request to stop all processes & exit")
	s.procMgr.StopAllProcesses()
	go func() {
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()
	return nil
}

func (s *Supervisor) Restart(args *struct{}, reply *rpcclient.StatusReply) error {
	log.Info("Receive instruction to restart")
	s.restarting = true
	reply.Success = true
	return nil
}

func (s *Supervisor) isRestarting() bool {
	return s.restarting
}

func getProcessInfo(proc *process.Process) *types.ProcessInfo {
	conf := proc.GetConfig().KeyValues()
	return &types.ProcessInfo{
		Name:          proc.GetName(),
		Group:         proc.GetGroup(),
		Description:   proc.GetDescription(),
		Start:         int(proc.GetStartTime().Unix()),
		Stop:          int(proc.GetStopTime().Unix()),
		Now:           int(time.Now().Unix()),
		State:         int(proc.GetState()),
		Statename:     proc.GetState().String(),
		Spawnerr:      "",
		Exitstatus:    proc.GetExitstatus(),
		Logfile:       proc.GetStdoutLogfile(),
		StdoutLogfile: proc.GetStdoutLogfile(),
		StderrLogfile: proc.GetStderrLogfile(),
		Pid:           proc.GetPid(),
		Directory:     conf["directory"],
		Command:       conf["command"],
		IniPath:       conf["ini_path"],
	}
}

func (s *Supervisor) GetAllProcessInfo(args *struct{}, reply *rpcclient.AllProcessInfoReply) error {
	reply.AllProcessInfo = make([]types.ProcessInfo, 0)
	s.procMgr.ForEachProcess(func(proc *process.Process) {
		procInfo := getProcessInfo(proc)
		reply.AllProcessInfo = append(reply.AllProcessInfo, *procInfo)
	})
	types.SortProcessInfos(reply.AllProcessInfo)
	return nil
}

func (s *Supervisor) GetProcessInfo(args *struct{ Name string }, reply *rpcclient.ProcessInfoReply) error {
	log.Info("Get process info of: ", args.Name)
	proc := s.procMgr.Find(args.Name)
	if proc == nil {
		return fmt.Errorf("no process named %s", args.Name)
	}

	reply.ProcessInfo = getProcessInfo(proc)
	return nil
}

func (s *Supervisor) StartProcess(args *StartProcessArgs, reply *rpcclient.StatusReply) error {
	if err := s.startProcess(args); err != nil {
		return errors.As(err)
	}
	reply.Success = true
	return nil
}

func (s *Supervisor) startProcess(args *StartProcessArgs) error {
	procs := s.procMgr.FindMatch(args.Name)
	if len(procs) <= 0 {
		return errors.New("fail to find process").As(args.Name)
	}
	for _, proc := range procs {
		proc.Start(args.Wait)
	}
	return nil
}

func (s *Supervisor) StartAllProcesses(args *struct {
	Wait bool `default:"true"`
}, reply *rpcclient.AllProcessInfoReply) error {
	ret, err := s.startAllProcesses(args.Wait)
	if err != nil {
		return errors.As(err)
	}
	reply.AllProcessInfo = ret
	return nil
}
func (s *Supervisor) startAllProcesses(wait bool) ([]types.ProcessInfo, error) {
	finishedProcCh := make(chan *process.Process)
	result := []types.ProcessInfo{}

	n := s.procMgr.AsyncForEachProcess(func(proc *process.Process) {
		proc.Start(wait)
	}, finishedProcCh)

	for i := 0; i < n; i++ {
		proc, ok := <-finishedProcCh
		if ok {
			processInfo := *getProcessInfo(proc)
			result = append(result, types.ProcessInfo{
				Name:        processInfo.Name,
				Group:       processInfo.Group,
				State:       faults.SUCCESS,
				Description: "OK",
			})
		}
	}
	return result, nil
}

func (s *Supervisor) StartProcessGroup(args *StartProcessArgs, reply *rpcclient.AllProcessInfoReply) error {
	log.WithFields(log.Fields{"group": args.Name}).Info("start process group")
	finishedProcCh := make(chan *process.Process)

	n := s.procMgr.AsyncForEachProcess(func(proc *process.Process) {
		if proc.GetGroup() == args.Name {
			proc.Start(args.Wait)
		}
	}, finishedProcCh)

	for i := 0; i < n; i++ {
		proc, ok := <-finishedProcCh
		if ok && proc.GetGroup() == args.Name {
			reply.AllProcessInfo = append(reply.AllProcessInfo, *getProcessInfo(proc))
		}
	}

	return nil
}

func (s *Supervisor) StopProcess(args *StartProcessArgs, reply *rpcclient.StatusReply) error {
	if err := s.stopProcess(args); err != nil {
		return errors.As(err)
	}
	reply.Success = true
	return nil
}
func (s *Supervisor) stopProcess(args *StartProcessArgs) error {
	log.WithFields(log.Fields{"program": args.Name}).Info("stop process")
	procs := s.procMgr.FindMatch(args.Name)
	if len(procs) <= 0 {
		return errors.New("fail to find process").As(args.Name)
	}
	for _, proc := range procs {
		proc.Stop(args.Wait)
	}
	return nil
}

func (s *Supervisor) StopProcessGroup(args *StartProcessArgs, reply *rpcclient.AllProcessInfoReply) error {
	log.WithFields(log.Fields{"group": args.Name}).Info("stop process group")
	finishedProcCh := make(chan *process.Process)
	n := s.procMgr.AsyncForEachProcess(func(proc *process.Process) {
		if proc.GetGroup() == args.Name {
			proc.Stop(args.Wait)
		}
	}, finishedProcCh)

	for i := 0; i < n; i++ {
		proc, ok := <-finishedProcCh
		if ok && proc.GetGroup() == args.Name {
			reply.AllProcessInfo = append(reply.AllProcessInfo, *getProcessInfo(proc))
		}
	}
	return nil
}

func (s *Supervisor) StopAllProcesses(args *struct {
	Wait bool `default:"true"`
}, reply *rpcclient.AllProcessInfoReply) error {
	ret, err := s.stopAllProcesses(args.Wait)
	if err != nil {
		return errors.As(err)
	}
	reply.AllProcessInfo = ret
	return nil
}
func (s *Supervisor) stopAllProcesses(wait bool) ([]types.ProcessInfo, error) {
	result := []types.ProcessInfo{}
	finishedProcCh := make(chan *process.Process)

	n := s.procMgr.AsyncForEachProcess(func(proc *process.Process) {
		proc.Stop(wait)
	}, finishedProcCh)

	for i := 0; i < n; i++ {
		proc, ok := <-finishedProcCh
		if ok {
			processInfo := *getProcessInfo(proc)
			result = append(result, types.ProcessInfo{
				Name:        processInfo.Name,
				Group:       processInfo.Group,
				State:       faults.SUCCESS,
				Description: "OK",
			})
		}
	}
	return result, nil
}

func (s *Supervisor) restartProcess(args *StartProcessArgs) error {
	if err := s.stopProcess(&StartProcessArgs{
		Name: args.Name,
		Wait: true,
	}); err != nil {
		return errors.As(err)
	}
	if err := s.startProcess(args); err != nil {
		return errors.As(err)
	}
	return nil
}
func (s *Supervisor) restartAllProcesses(wait bool) ([]types.ProcessInfo, error) {
	finishedProcCh := make(chan *process.Process)
	result := []types.ProcessInfo{}

	n := s.procMgr.AsyncForEachProcess(func(proc *process.Process) {
		proc.Stop(true)
		proc.Start(wait)
	}, finishedProcCh)

	for i := 0; i < n; i++ {
		proc, ok := <-finishedProcCh
		if ok {
			processInfo := *getProcessInfo(proc)
			result = append(result, types.ProcessInfo{
				Name:        processInfo.Name,
				Group:       processInfo.Group,
				State:       faults.SUCCESS,
				Description: "OK",
			})
		}
	}
	return result, nil
}

func (s *Supervisor) RestartProcess(args *StartProcessArgs, reply *rpcclient.StatusReply) error {
	if err := s.restartProcess(args); err != nil {
		return errors.As(err)
	}
	reply.Success = true
	return nil
}

func (s *Supervisor) RestartAllProcesses(args *struct {
	Wait bool `default:"true"`
}, reply *rpcclient.AllProcessInfoReply) error {
	ret, err := s.restartAllProcesses(args.Wait)
	if err != nil {
		return errors.As(err)
	}
	reply.AllProcessInfo = ret
	return nil
}

func (s *Supervisor) SignalProcess(args *types.ProcessSignal, reply *rpcclient.StatusReply) error {
	procs := s.procMgr.FindMatch(args.Name)
	if len(procs) <= 0 {
		reply.Success = false
		return fmt.Errorf("No process named %s", args.Name)
	}
	sig, err := signals.ToSignal(args.Signal)
	if err == nil {
		for _, proc := range procs {
			proc.Signal(sig, false)
		}
	}
	reply.Success = true
	return nil
}

func (s *Supervisor) SignalProcessGroup(args *types.ProcessSignal, reply *rpcclient.AllProcessInfoReply) error {
	s.procMgr.ForEachProcess(func(proc *process.Process) {
		if proc.GetGroup() == args.Name {
			sig, err := signals.ToSignal(args.Signal)
			if err == nil {
				proc.Signal(sig, false)
			}
		}
	})

	s.procMgr.ForEachProcess(func(proc *process.Process) {
		if proc.GetGroup() == args.Name {
			reply.AllProcessInfo = append(reply.AllProcessInfo, *getProcessInfo(proc))
		}
	})
	return nil
}

func (s *Supervisor) SignalAllProcesses(args *rpcclient.SignalAllProcessesArg, reply *rpcclient.SignalAllProcessesRet) error {
	s.procMgr.ForEachProcess(func(proc *process.Process) {
		sig, err := signals.ToSignal(args.Signal)
		if err == nil {
			proc.Signal(sig, false)
		}
	})
	s.procMgr.ForEachProcess(func(proc *process.Process) {
		reply.AllProcessInfo = append(reply.AllProcessInfo, *getProcessInfo(proc))
	})
	return nil
}

func (s *Supervisor) SendProcessStdin(args *ProcessStdin, reply *rpcclient.StatusReply) error {
	proc := s.procMgr.Find(args.Name)
	if proc == nil {
		log.WithFields(log.Fields{"program": args.Name}).Error("program does not exist")
		return fmt.Errorf("NOT_RUNNING")
	}
	if proc.GetState() != process.RUNNING {
		log.WithFields(log.Fields{"program": args.Name}).Error("program does not run")
		return fmt.Errorf("NOT_RUNNING")
	}
	err := proc.SendProcessStdin(args.Chars)
	if err == nil {
		reply.Success = true
	} else {
		reply.Success = false
	}
	return err
}

func (s *Supervisor) SendRemoteCommEvent(args *RemoteCommEvent, reply *rpcclient.StatusReply) error {
	events.EmitEvent(events.NewRemoteCommunicationEvent(args.Type, args.Data))
	reply.Success = true
	return nil
}

// return err, addedGroup, changedGroup, removedGroup
//
//
func (s *Supervisor) reload() (error, []string, []string, []string) {
	//get the previous loaded programs
	prevProgGroup := s.config.ProgramGroup.Clone()

	prevPrograms := s.config.ClonePrograms()
	prevProgramNames := []string{}
	for _, entry := range prevPrograms {
		prevProgramNames = append(prevProgramNames, entry.GetProgramName())
	}

	loaded_programs, err := s.config.Load()
	if err != nil {
		log.Warn(errors.As(err))
	} else {
		s.setSupervisordInfo()
		s.startEventListeners()
		s.createPrograms(prevProgramNames)
		s.startHttpServer()
	}

	// checking remove
	removedPrograms := util.Sub(prevProgramNames, loaded_programs)
	for _, removedProg := range removedPrograms {
		log.WithFields(log.Fields{"program": removedProg}).Info("the program is removed and will be stopped")
		s.config.RemoveProgram(removedProg)
		proc := s.procMgr.Remove(removedProg)
		if proc != nil {
			proc.Stop(false)
		}
	}

	// checking change
	curPrograms := s.config.GetPrograms()
	for _, pEntry := range prevPrograms {
		for _, cEntry := range curPrograms {
			name := pEntry.GetProgramName()
			if name != cEntry.GetProgramName() {
				continue
			}
			// Do reload
			proc := s.procMgr.Find(name)
			if proc == nil {
				log.WithFields(log.Fields{"program": name}).Info("the program not found")
				break
			}
			val1 := pEntry.String()
			val2 := cEntry.String()
			// not need to reload when value is same.
			if val1 == val2 {
				break
			}

			// try to reload configuration of the running process.
			log.WithFields(log.Fields{"program": name}).Info("the program reload by value changed")
			autoStart := proc.IsAutoStart()
			stoped := proc.Stoped()
			if !autoStart && stoped {
				// not auto start, and has stoped, not goto auto start.
				break
			}

			// do restart
			if !stoped {
				proc.Stop(true)
			}
			proc.Start(false)
			break
		}
	}

	// checking add
	addedPrograms := util.Sub(loaded_programs, prevProgramNames)
	for _, name := range addedPrograms {
		proc := s.procMgr.Find(name)
		if proc == nil {
			log.WithFields(log.Fields{"program": name}).Info("the program not found")
			break
		}
		if proc.IsAutoStart() {
			proc.Start(false)
		}
	}

	// TODO: value change for group
	addedGroup, changedGroup, removedGroup := s.config.ProgramGroup.Sub(prevProgGroup)
	return err, addedGroup, changedGroup, removedGroup
}

func (s *Supervisor) waitForExit() {
	for {
		if s.isRestarting() {
			s.procMgr.StopAllProcesses()
			break
		}
		time.Sleep(10 * time.Second)
	}
}

func (s *Supervisor) createPrograms(prevPrograms []string) {

	programs := s.config.GetProgramNames()
	for _, entry := range s.config.GetPrograms() {
		s.procMgr.CreateProcess(s.getSupervisorId(), entry)
	}
	removedPrograms := util.Sub(prevPrograms, programs)
	for _, p := range removedPrograms {
		s.procMgr.Remove(p)
	}
}

func (s *Supervisor) startEventListeners() {
	eventListeners := s.config.GetEventListeners()
	for _, entry := range eventListeners {
		proc := s.procMgr.CreateProcess(s.getSupervisorId(), entry)
		proc.Start(false)
	}

	if len(eventListeners) > 0 {
		time.Sleep(1 * time.Second)
	}
}

func (s *Supervisor) startHttpServer() {
	httpServerConfig, ok := s.config.GetInetHttpServer()
	s.rpcServer.Stop()
	if ok {
		addr := httpServerConfig.GetString("port", "")
		if addr != "" {
			go s.rpcServer.StartInetHttpServer(httpServerConfig.GetString("username", ""), httpServerConfig.GetString("password", ""), addr)
		}
	}

	httpServerConfig, ok = s.config.GetUnixHttpServer()
	if ok {
		env := config.NewStringExpression("here", s.config.GetConfigFileDir())
		sockFile, err := env.Eval(httpServerConfig.GetString("file", "/tmp/supervisord.sock"))
		if err == nil {
			go s.rpcServer.StartUnixHttpServer(httpServerConfig.GetString("username", ""), httpServerConfig.GetString("password", ""), sockFile)
		}
	}
}

func (s *Supervisor) setSupervisordInfo() {
	supervisordConf, ok := s.config.GetSupervisord()
	if ok {
		//set supervisord log
		env := config.NewStringExpression("here", s.config.GetConfigFileDir())
		logFile, err := env.Eval(supervisordConf.GetString("logfile", "supervisord.log"))
		if err != nil {
			logFile, err = process.Path_expand(logFile)
		}
		if logFile == "/dev/stdout" {
			return
		}
		logEventEmitter := logger.NewNullLogEventEmitter()
		s.logger = logger.NewNullLogger(logEventEmitter)
		if err == nil {
			logfile_maxbytes := int64(supervisordConf.GetBytes("logfile_maxbytes", 50*1024*1024))
			logfile_backups := supervisordConf.GetInt("logfile_backups", 10)
			loglevel := supervisordConf.GetString("loglevel", "info")
			s.logger, err = logger.NewLogger("supervisord", logFile, &sync.Mutex{}, logfile_maxbytes, logfile_backups, logEventEmitter)
			if err != nil {
				log.Fatal(errors.As(err))
			}
			log.SetLevel(toLogLevel(loglevel))
			log.SetOutput(s.logger)
		}
		//set the pid
		pidfile, err := env.Eval(supervisordConf.GetString("pidfile", "supervisord.pid"))
		if err == nil {
			if err := os.MkdirAll(filepath.Dir(pidfile), 0755); err != nil {
				log.Fatal(errors.As(err))
			}
			f, err := os.Create(pidfile)
			if err != nil {
				log.Fatal(errors.As(err))
			}
			fmt.Fprintf(f, "%d", os.Getpid())
			f.Close()
		}
	}
}

func toLogLevel(level string) log.Level {
	switch strings.ToLower(level) {
	case "critical":
		return log.FatalLevel
	case "error":
		return log.ErrorLevel
	case "warn":
		return log.WarnLevel
	case "info":
		return log.InfoLevel
	default:
		return log.DebugLevel
	}
}

func (s *Supervisor) ReloadConfig(args *rpcclient.ReloadConfigArg, reply *rpcclient.ReloadConfigRet) error {
	log.Info("start to reload config")
	err, addedGroup, changedGroup, removedGroup := s.reload()
	if len(addedGroup) > 0 {
		log.WithFields(log.Fields{"groups": strings.Join(addedGroup, ",")}).Info("added groups")
	}

	if len(changedGroup) > 0 {
		log.WithFields(log.Fields{"groups": strings.Join(changedGroup, ",")}).Info("changed groups")
	}

	if len(removedGroup) > 0 {
		log.WithFields(log.Fields{"groups": strings.Join(removedGroup, ",")}).Info("removed groups")
	}
	reply.AddedGroup = addedGroup
	reply.ChangedGroup = changedGroup
	reply.RemovedGroup = removedGroup
	return err
}

func (s *Supervisor) AddProcessGroup(args *struct{ Name string }, reply *rpcclient.StatusReply) error {
	reply.Success = false
	return nil
}

func (s *Supervisor) RemoveProcessGroup(args *struct{ Name string }, reply *rpcclient.StatusReply) error {
	reply.Success = false
	return nil
}
