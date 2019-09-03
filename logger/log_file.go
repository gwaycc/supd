package logger

import (
	"fmt"
	"os"
	"sync"

	"github.com/gwaycc/supd/faults"
	"github.com/gwaycc/supd/filebak"
	"github.com/gwaylib/errors"
)

type FileLogger struct {
	file            *filebak.File
	logEventEmitter LogEventEmitter
	locker          sync.Mutex
}

func NewFileLogger(name string, maxSize int64, backups int, logEventEmitter LogEventEmitter, locker sync.Locker) (*FileLogger, error) {
	file, err := filebak.OpenFile(name, maxSize, backups)
	if err != nil {
		return nil, errors.As(err, name)
	}

	logger := &FileLogger{
		file:            file,
		logEventEmitter: logEventEmitter,
	}
	return logger, nil
}

func (l *FileLogger) SetPid(pid int) {
	//NOTHING TO DO
}

// clear the current log file contents
func (l *FileLogger) ClearCurLogFile() error {
	return errors.New("Not implements")
}

func (l *FileLogger) ClearAllLogFile() error {
	return errors.New("Not implements")
}

func (l *FileLogger) ReadLog(offset int64, length int64) (string, error) {
	if offset < 0 && length != 0 {
		return "", faults.NewFault(faults.BAD_ARGUMENTS, "BAD_ARGUMENTS")
	}
	if offset >= 0 && length < 0 {
		return "", faults.NewFault(faults.BAD_ARGUMENTS, "BAD_ARGUMENTS")
	}

	l.locker.Lock()
	defer l.locker.Unlock()
	f, err := os.Open(l.file.Name())

	if err != nil {
		return "", faults.NewFault(faults.FAILED, "FAILED")
	}
	defer f.Close()

	//check the length of file
	statInfo, err := f.Stat()
	if err != nil {
		return "", faults.NewFault(faults.FAILED, "FAILED")
	}

	fileLen := statInfo.Size()

	if offset < 0 { //offset < 0 && length == 0
		offset = fileLen + offset
		if offset < 0 {
			offset = 0
		}
		length = fileLen - offset
	} else if length == 0 { //offset >= 0 && length == 0
		if offset > fileLen {
			return "", nil
		}
		length = fileLen - offset
	} else { //offset >= 0 && length > 0

		//if the offset exceeds the length of file
		if offset >= fileLen {
			return "", nil
		}

		//compute actual bytes should be read

		if offset+length > fileLen {
			length = fileLen - offset
		}
	}

	b := make([]byte, length)
	n, err := f.ReadAt(b, offset)
	if err != nil {
		return "", faults.NewFault(faults.FAILED, "FAILED")
	}
	return string(b[:n]), nil
}

func (l *FileLogger) ReadTailLog(offset int64, length int64) (string, int64, bool, error) {
	if offset < 0 {
		return "", offset, false, fmt.Errorf("offset should not be less than 0")
	}
	if length < 0 {
		return "", offset, false, fmt.Errorf("length should be not be less than 0")
	}
	l.locker.Lock()
	defer l.locker.Unlock()

	//open the file
	f, err := os.Open(l.file.Name())
	if err != nil {
		return "", 0, false, err
	}

	defer f.Close()

	//get the length of file
	statInfo, err := f.Stat()
	if err != nil {
		return "", 0, false, err
	}

	fileLen := statInfo.Size()

	//check if offset exceeds the length of file
	if offset >= fileLen {
		return "", fileLen, true, nil
	}

	//get the length
	if offset+length > fileLen {
		length = fileLen - offset
	}

	b := make([]byte, length)
	n, err := f.ReadAt(b, offset)
	if err != nil {
		return "", offset, false, err
	}
	return string(b[:n]), offset + int64(n), false, nil

}

// Override the function in io.Writer
func (l *FileLogger) Write(p []byte) (int, error) {
	l.logEventEmitter.emitLogEvent(string(p))
	return l.file.Write(p)
}

func (l *FileLogger) Close() error {
	return l.file.Close()
}
