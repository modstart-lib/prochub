package cli

import (
	"fmt"
	"io"
	"os"

	"prochub/internal/control"
)

// Options configures a CLI invocation.
type Options struct {
	DataDir string
	Version string
	Stdout  io.Writer
	Stderr  io.Writer
}

const usageText = `ProcHub 命令行工具

用法：
  prochub process list                        列出全部进程
  prochub process start <id>                  启动进程
  prochub process stop <id>                   停止进程
  prochub process logs <id> [选项]            查看进程日志
      --tail N                                仅显示最后 N 行（默认 100）
      --follow, -f                            持续输出新增日志
  prochub config get                          查看当前配置
  prochub config set theme <light|dark>       设置明暗主题
  prochub config set language <zh|en>         设置界面语言
  prochub config set autostart <on|off>       设置开机自动启动
  prochub config set autostart-wsl <on|off>   设置开机自动启动 WSL
  prochub status                              查看运行状态
  prochub version                             查看版本

说明：
  命令需要 ProcHub 正在运行。程序启动时会把控制端口与令牌写入数据目录的
  auth.json，命令行读取后与应用通信。
`

var cliCommands = map[string]bool{
	"process":   true,
	"config":    true,
	"status":    true,
	"version":   true,
	"help":      true,
	"-h":        true,
	"--help":    true,
	"-v":        true,
	"--version": true,
}

// IsCommand reports whether args begin with a recognized CLI command. When it
// returns false the caller should start the GUI as usual.
func IsCommand(args []string) bool {
	return len(args) > 0 && cliCommands[args[0]]
}

// Run executes a CLI command and returns a process exit code.
func Run(args []string, opts Options) int {
	if len(args) == 0 {
		fmt.Fprint(opts.Stderr, usageText)
		return 2
	}

	switch args[0] {
	case "help", "-h", "--help":
		fmt.Fprint(opts.Stdout, usageText)
		return 0
	case "version", "-v", "--version":
		fmt.Fprintf(opts.Stdout, "ProcHub %s\n", opts.Version)
		return 0
	}

	client, err := control.NewClient(opts.DataDir)
	if err != nil {
		fmt.Fprintln(opts.Stderr, connectError(err, opts.DataDir))
		return 1
	}

	var cmdErr error
	switch args[0] {
	case "status":
		cmdErr = runStatus(client, opts)
	case "process":
		cmdErr = runProcess(client, args[1:], opts)
	case "config":
		cmdErr = runConfig(client, args[1:], opts)
	default:
		fmt.Fprint(opts.Stderr, usageText)
		return 2
	}
	if cmdErr != nil {
		fmt.Fprintln(opts.Stderr, "错误："+cmdErr.Error())
		return 1
	}
	return 0
}

func connectError(err error, dataDir string) string {
	if os.IsNotExist(err) {
		return fmt.Sprintf("ProcHub 未运行（未找到 %s），请先启动 ProcHub", control.AuthPath(dataDir))
	}
	return fmt.Sprintf("无法连接 ProcHub：%v", err)
}
