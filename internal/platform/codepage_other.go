//go:build !windows

package platform

// activeCodePage has no meaning outside Windows, where process output is
// UTF-8 already. Returning 0 makes DecodeOutput keep the raw bytes for any
// non-UTF-8 input.
func activeCodePage() uint32 {
	return 0
}
