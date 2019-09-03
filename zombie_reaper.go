// +build !windows

package supd

import (
	reaper "github.com/ochinchina/go-reaper"
)

func ReapZombie() {
	go reaper.Reap()
}
