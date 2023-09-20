package localconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rongzer/blockchain/common/viperutil"
	"github.com/spf13/viper"
)

// Load 读取yaml文件及环境变量, 反序列化得到配置数据
func Load() (*TopLevel, error) {
	// 创建viper实例, 配置其要读取的路径及文件名
	if err := initViper("orderer"); err != nil {
		return nil, err
	}
	// 同时也获取相关的环境变量
	viper.SetEnvPrefix("ORDERER")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	// 读取数据
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("Error reading configuration: %s", err)
	}
	// 反序列化配置数据到结构中
	var uconf TopLevel
	if err := viperutil.EnhancedExactUnmarshalFromGlobal(&uconf); err != nil {
		return nil, fmt.Errorf("Error unmarshaling config into struct: %s", err)
	}
	// 如果缺少某些配置, 尽可能使用缺省值补上
	if err := uconf.setMissingValueToDefault(filepath.Dir(viper.ConfigFileUsed())); err != nil {
		return nil, err
	}

	return &uconf, nil
}

// 判断目录是否存在
func dirExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.IsDir()
}

// 传入p为相对路径返回拼接的绝对路径, 传入绝对路径则返回自身
func translatePath(base, p string) string {
	if filepath.IsAbs(p) {
		return p
	}

	return filepath.Join(base, p)
}

// 用绝对路径替换相对路径
func translatePathInPlace(base string, p *string) {
	*p = translatePath(base, *p)
}

// 初始化viper配置
func initViper(configName string) error {
	var altPath = os.Getenv("BLOCKCHAIN_CFG_PATH")
	if altPath != "" {
		// 如果设置了该环境变量, 则只认其值作为要读取的配置文件路径
		if !dirExists(altPath) {
			return fmt.Errorf("BLOCKCHAIN_CFG_PATH %s does not exist", altPath)
		}

		viper.AddConfigPath(altPath)
	} else {
		// 如果环境变量中未指定路径, 则优先使用程序执行路径作为配置文件路径
		viper.AddConfigPath("./")

		// 如果规约的配置文件路径存在, 也加入配置路径, 作为备选
		const officialConfigPath = "/etc/rongzer/blockchain"
		if dirExists(officialConfigPath) {
			viper.AddConfigPath(officialConfigPath)
		}
	}

	// 设置要读取的配置文件名, 名称不应该包含扩展名
	viper.SetConfigName(configName)

	return nil
}

// 将CA配置中的相对路径转为绝对路径
func translateCAs(configDir string, certificateAuthorities []string) []string {
	var results []string
	for _, ca := range certificateAuthorities {
		result := translatePath(configDir, ca)
		results = append(results, result)
	}
	return results
}
