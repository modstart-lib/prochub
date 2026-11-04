package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"prochub/internal/updater"
)

const defaultUpdateEndpoint = "https://open.tecmz.com/open_app/ProcHub/app_manager/updater"

// runUpdate implements `prochub update check|download`. It talks to the update
// backend directly and does not need the GUI to be running, which makes it the
// end-to-end test entry point for the release pipeline.
func runUpdate(args []string, opts Options) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprint(opts.Stdout, updateUsage)
		return 0
	}
	action := args[0]
	if action != "check" && action != "download" {
		fmt.Fprint(opts.Stderr, updateUsage)
		return 2
	}

	fs := flag.NewFlagSet("update "+action, flag.ContinueOnError)
	fs.SetOutput(opts.Stderr)
	endpoint := fs.String("endpoint", "", "update backend URL")
	channel := fs.String("channel", "latest", "release channel: latest|stable|beta")
	current := fs.String("current", opts.Version, "version to compare against")
	dir := fs.String("dir", "", "download directory")
	jsonOut := fs.Bool("json", false, "print the result as JSON")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if *endpoint == "" {
		*endpoint = os.Getenv("PROCHUB_UPDATE_ENDPOINT")
	}
	if *endpoint == "" {
		*endpoint = defaultUpdateEndpoint
	}
	if *dir == "" {
		*dir = filepath.Join(os.TempDir(), "prochub-update")
	}

	manager := updater.New(updater.Options{
		Endpoint:       *endpoint,
		CurrentVersion: *current,
		Channel:        *channel,
		DataDir:        *dir,
		Platform:       updater.CurrentPlatform(),
		UserAgent:      "ProcHub-CLI/" + opts.Version,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	state, err := manager.Check(ctx)
	if err != nil {
		fmt.Fprintln(opts.Stderr, "错误："+err.Error())
		return 1
	}
	if action == "download" {
		if state.State != updater.StateAvailable {
			printUpdateState(opts.Stdout, state, *jsonOut)
			return 0
		}
		state, err = manager.Download(ctx)
		if err != nil {
			fmt.Fprintln(opts.Stderr, "错误："+err.Error())
			return 1
		}
	}
	printUpdateState(opts.Stdout, state, *jsonOut)
	return 0
}

func printUpdateState(out io.Writer, state updater.State, asJSON bool) {
	if asJSON {
		data, _ := json.MarshalIndent(state, "", "  ")
		fmt.Fprintln(out, string(data))
		return
	}
	fmt.Fprintf(out, "当前版本: %s\n", state.CurrentVersion)
	if state.LatestVersion != "" {
		fmt.Fprintf(out, "最新版本: %s\n", state.LatestVersion)
	}
	fmt.Fprintf(out, "状态: %s\n", state.State)
	if state.Release != nil {
		if state.Release.Summary != "" {
			fmt.Fprintf(out, "更新说明: %s\n", state.Release.Summary)
		}
		if state.Release.Source != "" {
			fmt.Fprintf(out, "来源: %s\n", state.Release.Source)
		}
	}
	if state.Asset != nil {
		fmt.Fprintf(out, "下载资产: %s (%d bytes)\n", state.Asset.Name, state.Asset.Size)
		fmt.Fprintf(out, "sha256: %s\n", state.Asset.SHA256)
		for _, url := range state.Asset.URLs {
			fmt.Fprintf(out, "  镜像: %s\n", url)
		}
	}
	if state.State == updater.StateDownloading || state.State == updater.StateReady {
		fmt.Fprintf(out, "进度: %d%% (%d/%d)\n", state.Progress, state.Downloaded, state.Total)
	}
	if state.DownloadedPath != "" {
		fmt.Fprintf(out, "已下载: %s\n", state.DownloadedPath)
	}
	if state.Error != "" {
		fmt.Fprintf(out, "错误: %s\n", state.Error)
	}
}

const updateUsage = `更新命令

用法：
  prochub update check [选项]                 检查最新版本
  prochub update download [选项]              检查并下载、校验最新版本

选项：
  --endpoint URL      更新后端地址（默认取 PROCHUB_UPDATE_ENDPOINT）
  --channel NAME      latest / stable / beta（默认 latest，含 beta 打包）
  --current VERSION   当前版本（默认取本程序版本）
  --dir PATH          下载目录
  --json              以 JSON 输出
`
