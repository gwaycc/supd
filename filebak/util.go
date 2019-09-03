package filebak

import (
	"fmt"
	"strconv"

	"github.com/gwaylib/errors"
)

const (
	SIZE_K = 1024
	SIZE_M = 1024 * SIZE_K
	SIZE_G = 1024 * SIZE_M
)

func StrToSize(hum string) int64 {
	if len(hum) == 0 {
		return 0
	}
	if len(hum) < 2 {
		result, err := strconv.ParseInt(hum, 10, 64)
		if err != nil {
			panic(errors.As(err, hum))
		}
		return result
	}
	endIdx := len(hum) - 2
	end := string(hum[endIdx:])
	switch end {
	case "KB":
		result, err := strconv.ParseFloat(string(hum[:endIdx]), 64)
		if err != nil {
			panic(errors.As(err, hum))
		}
		return int64(result * SIZE_K)
	case "MB":
		result, err := strconv.ParseFloat(string(hum[:endIdx]), 64)
		if err != nil {
			panic(errors.As(err, hum))
		}
		return int64(result * SIZE_M)
	case "GB":
		result, err := strconv.ParseFloat(string(hum[:endIdx]), 64)
		if err != nil {
			panic(errors.As(err, hum))
		}
		return int64(result * SIZE_G)
	}
	panic("Unknow :" + end)
}

func SizeToStr(size int64) string {
	if size < SIZE_K {
		return fmt.Sprintf("%d", size)
	}
	if size < SIZE_M {
		return fmt.Sprintf("%.2fKB", float64(size)/float64(SIZE_K))
	}
	if size < SIZE_G {
		return fmt.Sprintf("%.2fMB", float64(size)/float64(SIZE_M))
	}

	return fmt.Sprintf("%.2fGB", float64(size)/float64(SIZE_G))
}
