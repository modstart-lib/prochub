package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"prochub/internal/control"
	"prochub/internal/logging"
	"prochub/internal/process"
)

func runProcess(client *control.Client, args []string, opts Options) error {
	if len(args) == 0 {
		return errors.New("用法：prochub process <list|start|stop|logs>")
	}
	switch args[0] {
	case "list":
		return processList(client, opts)
	case "start":
		return processStart(client, args[1:], opts)
	case "stop":
		return processStop(client, args[1:], opts)
	case "logs":
		return processLogs(client, args[1:], opts)
	default:
		return fmt.Errorf("未知的 process 子命令：%s", args[0])
	}
}

func processList(client *control.Client, opts Options) error {
	items, err := client.ListProcesses()
	if err != nil {
		return err
	}
	if len(items) == 0 {
		fmt.Fprintln(opts.Stdout, "暂无进程")
		return nil
	}
	for _, item := range items {
		name := item.Definition.Name
		if name == "" {
			name = item.Definition.ID
		}
		fmt.Fprintf(opts.Stdout, "%s  %s  [%s]  pid=%d  restarts=%d\n",
			item.Definition.ID, name, statusText(item.Status), item.PID, item.Restarts)
	}
	return nil
}

func processStart(client *control.Client, args []string, opts Options) error {
	if len(args) < 1 {
		return errors.New("请指定进程 ID：prochub process start <id>")
	}
	id := args[0]
	if err := client.StartProcess(id); err != nil {
		return err
	}
	fmt.Fprintf(opts.Stdout, "已启动进程：%s\n", id)
	return nil
}

func processStop(client *control.Client, args []string, opts Options) error {
	if len(args) < 1 {
		return errors.New("请指定进程 ID：prochub process stop <id>")
	}
	id := args[0]
	if err := client.StopProcess(id); err != nil {
		return err
	}
	fmt.Fprintf(opts.Stdout, "已停止进程：%s\n", id)
	return nil
}

func processLogs(client *control.Client, args []string, opts Options) error {
	fs := flag.NewFlagSet("process logs", flag.ContinueOnError)
	fs.SetOutput(opts.Stderr)
	tail := fs.Int("tail", 100, "仅显示最后 N 行")
	follow := fs.Bool("follow", false, "持续输出新增日志")
	fs.BoolVar(follow, "f", false, "持续输出新增日志")
	if err := fs.Parse(args); err != nil {
		return errors.New("用法：prochub process logs <id> [--tail N] [--follow]")
	}

	rest := fs.Args()
	if len(rest) < 1 {
		return errors.New("请指定进程 ID：prochub process logs <id>")
	}
	id := rest[0]

	lines, err := client.Logs(id, *tail)
	if err != nil {
		return err
	}
	for _, line := range lines {
		fmt.Fprintln(opts.Stdout, line)
	}
	if !*follow {
		return nil
	}

	cfg, err := client.Config()
	if err != nil {
		return err
	}
	tailer := logging.NewTailer(filepath.Join(opts.DataDir, cfg.LogDir, id))
	if err := tailer.SeekEnd(); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			newLines, err := tailer.Next()
			if err != nil {
				return err
			}
			for _, line := range newLines {
				fmt.Fprintln(opts.Stdout, line)
			}
		}
	}
}

func statusText(status process.Status) string {
	switch status {
	case process.StatusRunning:
		return "运行中"
	case process.StatusStarting:
		return "启动中"
	case process.StatusErrored:
		return "异常"
	default:
		return "已停止"
	}
}
