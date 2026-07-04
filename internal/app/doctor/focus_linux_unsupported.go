//go:build !linux

package doctor

func detectLinuxFocusSupport() bool {
	return false
}
