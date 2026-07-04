//go:build !windows

package doctor

func detectWindowsFocusHelper() bool {
	return false
}
