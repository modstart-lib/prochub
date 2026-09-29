package cli

import (
	"errors"
	"fmt"

	"prochub/internal/control"
)

func runConfig(client *control.Client, args []string, opts Options) error {
	if len(args) == 0 {
		return errors.New("用法：prochub config <get|set>")
	}
	switch args[0] {
	case "get":
		return configGet(client, opts)
	case "set":
		return configSet(client, args[1:], opts)
	default:
		return fmt.Errorf("未知的 config 子命令：%s", args[0])
	}
}

func configGet(client *control.Client, opts Options) error {
	cfg, err := client.Config()
	if err != nil {
		return err
	}
	fmt.Fprintf(opts.Stdout, "主题: %s\n", cfg.Theme)
	fmt.Fprintf(opts.Stdout, "语言: %s\n", cfg.Locale)
	fmt.Fprintf(opts.Stdout, "开机自动启动: %s\n", onOffText(cfg.AutoStart))
	fmt.Fprintf(opts.Stdout, "开机自动启动 WSL: %s\n", onOffText(cfg.AutoStartWSL))
	return nil
}

func configSet(client *control.Client, args []string, opts Options) error {
	if len(args) < 2 {
		return errors.New("用法：prochub config set <theme|language|autostart|autostart-wsl> <value>")
	}
	key, value := args[0], args[1]
	patch := control.ConfigPatch{}
	display := ""

	switch key {
	case "theme":
		v, err := parseTheme(value)
		if err != nil {
			return err
		}
		patch.Theme = &v
		display = "主题：" + v
	case "language":
		v, err := parseLocale(value)
		if err != nil {
			return err
		}
		patch.Locale = &v
		display = "语言：" + v
	case "autostart":
		v, err := parseOnOff(value)
		if err != nil {
			return err
		}
		patch.AutoStart = &v
		display = "开机自动启动：" + onOffText(v)
	case "autostart-wsl":
		v, err := parseOnOff(value)
		if err != nil {
			return err
		}
		patch.AutoStartWSL = &v
		display = "开机自动启动 WSL：" + onOffText(v)
	default:
		return fmt.Errorf("未知配置项：%s（支持 theme / language / autostart / autostart-wsl）", key)
	}

	if err := client.UpdateConfig(patch); err != nil {
		return err
	}
	fmt.Fprintf(opts.Stdout, "已更新%s\n", display)
	return nil
}

func parseTheme(value string) (string, error) {
	switch value {
	case "light", "dark":
		return value, nil
	}
	return "", fmt.Errorf("主题仅支持 light 或 dark，收到：%s", value)
}

func parseLocale(value string) (string, error) {
	switch value {
	case "zh", "en":
		return value, nil
	}
	return "", fmt.Errorf("语言仅支持 zh 或 en，收到：%s", value)
}

func parseOnOff(value string) (bool, error) {
	switch value {
	case "on", "true", "1", "enable", "enabled":
		return true, nil
	case "off", "false", "0", "disable", "disabled":
		return false, nil
	}
	return false, fmt.Errorf("开关仅支持 on 或 off，收到：%s", value)
}

func onOffText(v bool) string {
	if v {
		return "已开启"
	}
	return "已关闭"
}
