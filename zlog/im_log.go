package zlog

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

func NewLogger(logPath string) {

	zapConfig := zap.Config{
		Encoding: "json",
		Level:    zap.NewAtomicLevelAt(zapcore.DebugLevel),
		// OutputPaths: []string{"stdout"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:  "message",
			LevelKey:    "level",
			EncodeLevel: zapcore.LowercaseLevelEncoder, // INFO

			TimeKey:    "time",
			EncodeTime: zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000000"),

			CallerKey: "caller",
			EncodeCaller: func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
				// TODO: consider using a byte-oriented API to save an allocation.
				enc.AppendString(trimmedPath(caller))
				// enc.AppendString(caller.TrimmedPath())
			},
		},
	}

	options := []zap.Option{
		zap.AddCallerSkip(1),                             // 上一层日志调用路径
		zap.AddStacktrace(zap.WarnLevel),                 // 设置warn/error级别的日志会输出堆栈调用
		logSplitOption(logPath, zapConfig.EncoderConfig), // 日志分割,不同输出方式
	}

	var err error
	logger, err = zapConfig.Build(options...)
	if err != nil {
		panic(err)
	}
}

func logSplitOption(logPath string, encoderConfig zapcore.EncoderConfig) zap.Option {

	lumberJackLogger := &DivisionLogger{
		Filename:         logPath,
		MaxSize:          1,     // 切割之前文件最大大小(单位:MB)
		MaxBackups:       100,   // 保留旧文件最大个数
		MaxAge:           1,     // 保留旧文件最大天数
		Compress:         false, // 是否压缩/归档旧文件
		LocalTime:        false,
		BackupTimeFormat: "2006-01-02",
	}

	encoder := zapcore.NewJSONEncoder(encoderConfig)

	cores := []zapcore.Core{
		zapcore.NewCore(encoder, zapcore.AddSync(lumberJackLogger), zapcore.DebugLevel),
		zapcore.NewCore(encoder, zapcore.Lock(os.Stdout), zapcore.DebugLevel),
	}

	return zap.WrapCore(func(zapcore.Core) zapcore.Core {
		return zapcore.NewTee(cores...)
	})
}

func Info(v ...interface{}) {
	info := fmt.Sprintf(v[0].(string), v[1:]...)
	logger.Info(info)
}

func Warn(v ...interface{}) {
	info := fmt.Sprintf(v[0].(string), v[1:]...)
	logger.Warn(info)
}

func Error(v ...interface{}) {
	info := fmt.Sprintf(v[0].(string), v[1:]...)
	logger.Error(info)
}
