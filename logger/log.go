package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/gwaycc/supd/events"
	"github.com/gwaycc/supd/faults"
	"github.com/gwaylib/errors"
)

//implements io.Writer interface

type Logger interface {
	io.WriteCloser
	SetPid(pid int)
	ReadLog(offset int64, length int64) (string, error)
	ReadTailLog(offset int64, length int64) (string, int64, bool, error)
	ClearCurLogFile() error
	ClearAllLogFile() error
}

type LogEventEmitter interface {
	emitLogEvent(data string)
}

type SysLogger struct {
	NullLogger
	logWriter       io.WriteCloser
	logEventEmitter LogEventEmitter
}

type NullLogger struct {
	logEventEmitter LogEventEmitter
}

type NullLocker struct {
}

type CompositeLogger struct {
	loggers []Logger
}

func (sl *SysLogger) Write(b []byte) (int, error) {
	sl.logEventEmitter.emitLogEvent(string(b))
	if sl.logWriter == nil {
		return 0, errors.New("not connect to syslog server")
	}
	return sl.logWriter.Write(b)
}

func (sl *SysLogger) Close() error {
	if sl.logWriter == nil {
		return errors.New("not connect to syslog server")
	}
	return sl.logWriter.Close()
}
func NewNullLogger(logEventEmitter LogEventEmitter) *NullLogger {
	return &NullLogger{logEventEmitter: logEventEmitter}
}

func (l *NullLogger) SetPid(pid int) {
	//NOTHING TO DO
}

func (l *NullLogger) Write(p []byte) (int, error) {
	l.logEventEmitter.emitLogEvent(string(p))
	return len(p), nil
}

func (l *NullLogger) Close() error {
	return nil
}

func (l *NullLogger) ReadLog(offset int64, length int64) (string, error) {
	return "", faults.NewFault(faults.NO_FILE, "NO_FILE")
}

func (l *NullLogger) ReadTailLog(offset int64, length int64) (string, int64, bool, error) {
	return "", 0, false, faults.NewFault(faults.NO_FILE, "NO_FILE")
}

func (l *NullLogger) ClearCurLogFile() error {
	return fmt.Errorf("No log")
}

func (l *NullLogger) ClearAllLogFile() error {
	return faults.NewFault(faults.NO_FILE, "NO_FILE")
}

func NewNullLocker() *NullLocker {
	return &NullLocker{}
}

func (l *NullLocker) Lock() {
}

func (l *NullLocker) Unlock() {
}

type StdLogger struct {
	NullLogger
	logEventEmitter LogEventEmitter
	writer          io.Writer
}

func NewStdoutLogger(logEventEmitter LogEventEmitter) *StdLogger {
	return &StdLogger{logEventEmitter: logEventEmitter,
		writer: os.Stdout}
}

func (l *StdLogger) Write(p []byte) (int, error) {
	n, err := l.writer.Write(p)
	if err != nil {
		l.logEventEmitter.emitLogEvent(string(p))
	}
	return n, err
}

func NewStderrLogger(logEventEmitter LogEventEmitter) *StdLogger {
	return &StdLogger{logEventEmitter: logEventEmitter,
		writer: os.Stderr}
}

type LogCaptureLogger struct {
	underlineLogger        Logger
	procCommEventCapWriter io.Writer
	procCommEventCapture   *events.ProcCommEventCapture
}

func NewLogCaptureLogger(underlineLogger Logger,
	captureMaxBytes int,
	stdType string,
	procName string,
	groupName string) *LogCaptureLogger {
	r, w := io.Pipe()
	eventCapture := events.NewProcCommEventCapture(r,
		captureMaxBytes,
		stdType,
		procName,
		groupName)
	return &LogCaptureLogger{underlineLogger: underlineLogger,
		procCommEventCapWriter: w,
		procCommEventCapture:   eventCapture}
}

func (l *LogCaptureLogger) SetPid(pid int) {
	l.procCommEventCapture.SetPid(pid)
}

func (l *LogCaptureLogger) Write(p []byte) (int, error) {
	l.procCommEventCapWriter.Write(p)
	return l.underlineLogger.Write(p)
}

func (l *LogCaptureLogger) Close() error {
	return l.underlineLogger.Close()
}

func (l *LogCaptureLogger) ReadLog(offset int64, length int64) (string, error) {
	return l.underlineLogger.ReadLog(offset, length)
}

func (l *LogCaptureLogger) ReadTailLog(offset int64, length int64) (string, int64, bool, error) {
	return l.underlineLogger.ReadTailLog(offset, length)
}

func (l *LogCaptureLogger) ClearCurLogFile() error {
	return l.underlineLogger.ClearCurLogFile()
}

func (l *LogCaptureLogger) ClearAllLogFile() error {
	return l.underlineLogger.ClearAllLogFile()
}

type NullLogEventEmitter struct {
}

func NewNullLogEventEmitter() *NullLogEventEmitter {
	return &NullLogEventEmitter{}
}

func (ne *NullLogEventEmitter) emitLogEvent(data string) {
}

type StdLogEventEmitter struct {
	Type         string
	process_name string
	group_name   string
	pidFunc      func() int
}

func NewStdoutLogEventEmitter(process_name string, group_name string, procPidFunc func() int) *StdLogEventEmitter {
	return &StdLogEventEmitter{Type: "stdout",
		process_name: process_name,
		group_name:   group_name,
		pidFunc:      procPidFunc}
}

func NewStderrLogEventEmitter(process_name string, group_name string, procPidFunc func() int) *StdLogEventEmitter {
	return &StdLogEventEmitter{Type: "stderr",
		process_name: process_name,
		group_name:   group_name,
		pidFunc:      procPidFunc}
}

func (se *StdLogEventEmitter) emitLogEvent(data string) {
	if se.Type == "stdout" {
		events.EmitEvent(events.CreateProcessLogStdoutEvent(se.process_name, se.group_name, se.pidFunc(), data))
	} else {
		events.EmitEvent(events.CreateProcessLogStderrEvent(se.process_name, se.group_name, se.pidFunc(), data))
	}
}

type BackgroundWriteCloser struct {
	io.WriteCloser
	logChannel  chan []byte
	writeCloser io.WriteCloser
}

func NewBackgroundWriteCloser(writeCloser io.WriteCloser) *BackgroundWriteCloser {
	channel := make(chan []byte)
	bw := &BackgroundWriteCloser{logChannel: channel,
		writeCloser: writeCloser}

	bw.start()
	return bw
}

func (bw *BackgroundWriteCloser) start() {
	go func() {
		for {
			b, ok := <-bw.logChannel
			if !ok {
				break
			}
			bw.writeCloser.Write(b)
		}
	}()
}

func (bw *BackgroundWriteCloser) Write(p []byte) (n int, err error) {
	bw.logChannel <- p
	return len(p), nil
}

func (bw *BackgroundWriteCloser) Close() error {
	close(bw.logChannel)
	return bw.writeCloser.Close()
}

func NewCompositeLogger(loggers []Logger) *CompositeLogger {
	return &CompositeLogger{loggers: loggers}
}

func (cl *CompositeLogger) Write(p []byte) (n int, err error) {
	for i, logger := range cl.loggers {
		if i == 0 {
			n, err = logger.Write(p)
		} else {
			logger.Write(p)
		}
	}
	return
}

func (cl *CompositeLogger) Close() (err error) {
	for i, logger := range cl.loggers {
		if i == 0 {
			err = logger.Close()
		} else {
			logger.Close()
		}
	}
	return
}

func (cl *CompositeLogger) SetPid(pid int) {
	for _, logger := range cl.loggers {
		logger.SetPid(pid)
	}
}

func (cl *CompositeLogger) ReadLog(offset int64, length int64) (string, error) {
	return cl.loggers[0].ReadLog(offset, length)
}

func (cl *CompositeLogger) ReadTailLog(offset int64, length int64) (string, int64, bool, error) {
	return cl.loggers[0].ReadTailLog(offset, length)
}

func (cl *CompositeLogger) ClearCurLogFile() error {
	return cl.loggers[0].ClearCurLogFile()
}

func (cl *CompositeLogger) ClearAllLogFile() error {
	return cl.loggers[0].ClearAllLogFile()
}

// create a logger for a program with parameters
//
func NewLogger(programName string, logFile string, locker sync.Locker, maxBytes int64, backups int, logEventEmitter LogEventEmitter) (Logger, error) {
	files := splitLogFile(logFile)
	loggers := make([]Logger, 0)
	for i, f := range files {
		var lr Logger
		var err error
		if i == 0 {
			lr, err = createLogger(programName, f, locker, maxBytes, backups, logEventEmitter)
			if err != nil {
				return nil, errors.As(err)
			}
		} else {
			lr, err = createLogger(programName, f, NewNullLocker(), maxBytes, backups, NewNullLogEventEmitter())
			if err != nil {
				return nil, errors.As(err)
			}
		}
		loggers = append(loggers, lr)
	}
	if len(loggers) > 1 {
		return NewCompositeLogger(loggers), nil
	} else {
		return loggers[0], nil
	}
}

func splitLogFile(logFile string) []string {
	files := strings.Split(logFile, ",")
	for i, f := range files {
		files[i] = strings.TrimSpace(f)
	}
	return files
}

func createLogger(programName string, logFile string, locker sync.Locker, maxBytes int64, backups int, logEventEmitter LogEventEmitter) (Logger, error) {
	if logFile == "/dev/stdout" {
		return NewStdoutLogger(logEventEmitter), nil
	}
	if logFile == "/dev/stderr" {
		return NewStderrLogger(logEventEmitter), nil
	}
	if logFile == "/dev/null" {
		return NewNullLogger(logEventEmitter), nil
	}

	if logFile == "syslog" {
		return NewSysLogger(programName, logEventEmitter), nil
	}
	if strings.HasPrefix(logFile, "syslog") {
		fields := strings.Split(logFile, "@")
		fields[0] = strings.TrimSpace(fields[0])
		fields[1] = strings.TrimSpace(fields[1])
		if len(fields) == 2 && fields[0] == "syslog" {
			return NewRemoteSysLogger(programName, fields[1], logEventEmitter), nil
		}
	}
	if len(logFile) > 0 {
		return NewFileLogger(logFile, maxBytes, backups, logEventEmitter, locker)
	}
	return NewNullLogger(logEventEmitter), nil
}
