//go:build windows

package boot

import (
    "os"

    "golang.org/x/sys/windows"
)

func init() {
    // 启用 Windows 虚拟终端处理以支持 ANSI 颜色码
    stdout := windows.Handle(os.Stdout.Fd())
    var mode uint32
    if err := windows.GetConsoleMode(stdout, &mode); err == nil {
        windows.SetConsoleMode(stdout, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
    }

    stderr := windows.Handle(os.Stderr.Fd())
    if err := windows.GetConsoleMode(stderr, &mode); err == nil {
        windows.SetConsoleMode(stderr, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
    }
}
