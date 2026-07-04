//go:build windows

package doctor

import "github.com/hellolib/toast"

func detectWindowsFocusHelper() bool {
	_, err := toast.FindFocusHelper()
	return err == nil
}
