//go:build linux

package doctor

import (
	"os"
)

func detectLinuxFocusSupport() bool {
	if os.Getenv("DISPLAY") == "" {
		return false
	}
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") != "" {
		return true
	}
	if uid := os.Getuid(); uid >= 0 {
		if info, err := os.Stat("/run/user/" + itoa(uid) + "/bus"); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for value > 0 {
		i--
		buf[i] = byte('0' + value%10)
		value /= 10
	}
	return string(buf[i:])
}
