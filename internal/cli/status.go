package cli

import (
	"fmt"

	"prochub/internal/control"
)

func runStatus(client *control.Client, opts Options) error {
	info, err := client.Status()
	if err != nil {
		return err
	}
	fmt.Fprintf(opts.Stdout, "ProcHub %s（%s） 已连接\n", info.Version, info.Platform)
	fmt.Fprintf(opts.Stdout, "进程：共 %d 个，运行中 %d 个\n", info.ProcessCount, info.RunningCount)
	fmt.Fprintf(opts.Stdout, "数据目录：%s\n", info.DataDir)
	return nil
}
