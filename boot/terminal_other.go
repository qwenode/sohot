//go:build !windows

package boot

// SetConsoleTitle 非 Windows 平台的空实现
func SetConsoleTitle(title string) {
	// 非 Windows 平台不需要设置控制台标题
}
