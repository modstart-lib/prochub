package cli

import (
	"errors"
	"fmt"

	"prochub/internal/control"
)

func runWSL(client *control.Client, args []string, opts Options) error {
	if len(args) == 0 {
		return errors.New("用法：prochub wsl <status|start|stop|restart>")
	}
	switch args[0] {
	case "status":
		return wslStatus(client, opts)
	case "start":
		return wslAction(client, "start", "已启动 WSL", opts)
	case "stop":
		return wslAction(client, "stop", "已停止 WSL", opts)
	case "restart":
		return wslAction(client, "restart", "已重启 WSL", opts)
	default:
		return fmt.Errorf("未知的 wsl 子命令：%s", args[0])
	}
}

func wslStatus(client *control.Client, opts Options) error {
	status, err := client.WSLStatus()
	if err != nil {
		return err
	}
	if !status.Supported {
		fmt.Fprintln(opts.Stdout, "当前平台不支持 WSL")
		return nil
	}
	fmt.Fprintf(opts.Stdout, "运行状态: %s\n", wslStateText(status))
	fmt.Fprintf(opts.Stdout, "默认发行版: %s\n", fallbackText(status.Distro, "未知"))
	fmt.Fprintf(opts.Stdout, "WSL 版本: %s\n", fallbackText(status.Version, "未知"))
	return nil
}

func wslAction(client *control.Client, action, success string, opts Options) error {
	var err error
	switch action {
	case "start":
		err = client.StartWSL()
	case "stop":
		err = client.StopWSL()
	case "restart":
		err = client.RestartWSL()
	}
	if err != nil {
		return err
	}
	fmt.Fprintln(opts.Stdout, success)
	return nil
}

func wslStateText(status control.WSLStatusInfo) string {
	switch {
	case status.Starting:
		return "启动中"
	case status.Running:
		return "运行中"
	case !status.Available:
		return "不可用（未安装或未安装发行版）"
	default:
		return "已停止"
	}
}

func fallbackText(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
