package filebak

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/gwaylib/errors"
)

type File struct {
	*os.File

	mutex   sync.Mutex
	bakNum  int // default: 10
	curSize int64
	maxSize int64 // default: 1M
}

func (lf *File) move(a, b string) error {
	if err := os.Rename(a, b); err != nil {
		if !strings.HasSuffix(err.Error(), "no such file or directory") {
			return errors.As(err, a)
		}
	}
	return nil
}

// Rewrite
func (lf *File) Write(data []byte) (int, error) {
	lf.mutex.Lock()
	defer lf.mutex.Unlock()

	n, err := lf.File.Write(data)
	if err != nil {
		return n, err
	}
	lf.curSize += int64(n)

	// 文件未到限量值
	if lf.curSize < lf.maxSize {
		return n, err
	}
	// 文件到达限量值, 进行文件处理
	if err := lf.Close(); err != nil {
		return n, errors.As(err)
	}

	oriName := lf.Name()
	// 删除最后一个文件
	if err := os.RemoveAll(fmt.Sprintf("%s.%d", oriName, lf.bakNum)); err != nil {
		return n, errors.As(err)
	}
	// 重命名文件
	for i := lf.bakNum - 1; i > 0; i-- {
		a := fmt.Sprintf("%s.%d", oriName, i)
		b := fmt.Sprintf("%s.%d", oriName, i+1)
		if err := lf.move(a, b); err != nil {
			return n, errors.As(err)
		}
	}
	// 备份文件
	if err := lf.move(oriName, oriName+".1"); err != nil {
		return n, errors.As(err)
	}

	// 重打开文件
	file, err := os.OpenFile(lf.Name(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return n, errors.As(err, lf.Name())
	}
	lf.File = file
	lf.curSize = 0 // clean size

	return n, nil
}

func (lf *File) WriteAt(b []byte, off int64) (n int, err error) {
	n, err = lf.File.WriteAt(b, off)
	if err != nil {
		return n, errors.As(err)
	}
	if lf.curSize < (off + int64(n)) {
		lf.curSize = off + int64(n)
	}
	return n, nil
}

func (lf *File) Truncate(size int64) error {
	if err := lf.File.Truncate(size); err != nil {
		return errors.As(err)
	}
	lf.curSize = size
	return nil
}

func (lf *File) WriteString(s string) (n int, err error) {
	return lf.Write([]byte(s))
}

func OpenFile(fileName string, maxSize int64, bakNum int) (*File, error) {
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		if strings.HasSuffix(err.Error(), "no such file or directory") {
			// Make directory, and retry
			if err := os.MkdirAll(path.Dir(fileName), 0755); err != nil {
				return nil, errors.As(err)
			}
			return OpenFile(fileName, maxSize, bakNum)
		}
		return nil, errors.As(err, fileName)
	}
	fileStat, err := file.Stat()
	if err != nil {
		return nil, errors.As(err, fileName)
	}
	return &File{
		File:    file,
		curSize: fileStat.Size(),
		maxSize: maxSize,
		bakNum:  bakNum,
	}, nil
}
