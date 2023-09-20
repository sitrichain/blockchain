package metadata

import (
	"runtime"

	"github.com/rongzer/blockchain/common/log"
)

// 在Makefile中通过ldflags注入的值
var Version string             // 版本
var BuildTime string           // 编译时间
var GitVersion string          // Git版本
var BaseVersion string         // 基础镜像版本
var BaseDockerLabel string     // 基础镜像名
var DockerNamespace string     // 镜像仓库名
var BaseDockerNamespace string // 基础镜像仓库名

// PrintVersionInfo 打印当前程序版本信息
func PrintVersionInfo() {
	if Version == "" {
		Version = "development build"
	}

	log.Logger.Infow("Runtime info",
		"Version", Version,
		"GitVersion", GitVersion,
		"BuildTime", BuildTime,
		"Go", runtime.Version(),
		"OS", runtime.GOOS,
		"Arch", runtime.GOARCH,
	)
}
