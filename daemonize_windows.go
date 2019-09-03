// +build windows

package supd

func Deamonize(proc func()) {
	proc()
}
