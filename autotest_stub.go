//go:build !autotest

package main

import "context"

// startAutotestServer 为空实现：开源版（不带 autotest tag）不启动测试服务。
// 该方法保证 app.go 在两种构建下都能编译通过。
func startAutotestServer(_ context.Context, _ *App) {}
