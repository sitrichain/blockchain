package log

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/arthurkiller/rollingwriter"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 全局日志对象
var Logger *zap.SugaredLogger

// 最低输出级别
var lowestLevel zapcore.Level

func init() {
	logger, _ := zap.NewDevelopment()
	Logger = logger.Sugar()
	lowestLevel = zap.InfoLevel
}

// InitLogger 初始化日志
func InitLogger(version, path, name, level string) error {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lowestLevel = zapcore.DebugLevel
	case "info":
		lowestLevel = zapcore.InfoLevel
	case "warn":
		lowestLevel = zapcore.WarnLevel
	case "error":
		lowestLevel = zapcore.ErrorLevel
	default:
		lowestLevel = zapcore.InfoLevel
	}

	infoLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= lowestLevel && lvl < zapcore.ErrorLevel
	})
	errorLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})

	infoWriter, err := newRotateWriter(path, name+"Access")
	if err != nil {
		return err
	}
	errorWriter, err := newRotateWriter(path, name+"Error")
	if err != nil {
		return err
	}
	encoderConfig := zap.NewProductionEncoderConfig()
	var rbcTimeEncoder zapcore.TimeEncoder
	rbcTimeEncoder = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
	}
	encoderConfig.EncodeTime = rbcTimeEncoder
	jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)
	cores := []zapcore.Core{
		zapcore.NewCore(jsonEncoder, zapcore.AddSync(infoWriter), infoLevel),
		zapcore.NewCore(jsonEncoder, zapcore.AddSync(errorWriter), errorLevel),
	}

	if !strings.Contains(version, "release") {
		allLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= lowestLevel
		})
		consoleEncoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
			TimeKey:        "T",
			LevelKey:       "L",
			NameKey:        "N",
			CallerKey:      "C",
			MessageKey:     "M",
			StacktraceKey:  "S",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		})
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), allLevel))
	}

	zapLogger := zap.New(zapcore.NewTee(cores...), zap.AddCaller())
	Logger = zapLogger.Sugar()
	return nil
}

// IsEnabledFor 是否打印判断
func IsEnabledFor(lvl zapcore.Level) bool {
	return lvl >= lowestLevel
}

// SyncLog 同步日志
func SyncLog() {
	if Logger != nil {
		if err := Logger.Sync(); err != nil {
			Logger.Error("Sync log error", zap.Error(err))
		}
	}
}

// 设置日志分不同文件保存, 按时间切割保存
func newRotateWriter(logPath, name string) (io.Writer, error) {
	if err := os.MkdirAll(logPath, os.ModePerm); err != nil {
		return nil, err
	}

	config := rollingwriter.Config{
		LogPath:       logPath,      // 日志路径
		TimeTagFormat: "2006-01-02", // 时间格式串
		FileName:      name,         // 日志文件名
		MaxRemain:     30,           // 配置日志最大存留数

		// 目前有2中滚动策略: 按照时间滚动按照大小滚动
		// - 时间滚动: 配置策略如同 crontable, 例如,每天0:0切分, 则配置 0 0 0 * * *
		// - 大小滚动: 配置单个日志文件(未压缩)的滚动大小门限, 入1G, 500M
		RollingPolicy:      rollingwriter.TimeRolling, // 配置滚动策略 norolling timerolling volumerolling
		RollingTimePattern: "0 0 0 * * *",             // 配置时间滚动策略
		RollingVolumeSize:  "1k",                      // 配置截断文件下限大小

		// writer 支持4种不同的 mode:
		// 1. none 2. lock
		// 3. async 4. buffer
		// - 无保护的 writer: 不提供并发安全保障
		// - lock 保护的 writer: 提供由 mutex 保护的并发安全保障
		// - 异步 writer: 异步 write, 并发安全. 异步开启后忽略 Lock 选项
		WriterMode: "lock",
		// BufferWriterThershould in B
		BufferWriterThershould: 8 * 1024 * 1024,
		// Compress will compress log file with gzip
		Compress: false,
	}

	// 创建一个 writer
	writer, err := rollingwriter.NewWriterFromConfig(&config)
	if err != nil {
		return nil, err
	}

	return writer, nil
}
