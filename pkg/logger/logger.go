package logger

import (
	"crypto_trading_system/pkg/config"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
	config *config.Config
}

type Fields map[string]interface{}

func NewLogger(config *config.Config) *Logger {
	logger := logrus.New()

	level, err := logrus.ParseLevel(config.Log.Level)
	if err != nil {
		level = logrus.InfoLevel
	}

	logger.SetLevel(level)

	if config.Log.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05.000",
			PrettyPrint:     false,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05.000",
			FullTimestamp:   true,
			DisableColors:   config.Log.Output == "file",
		})
	}

	// logger.SetOutput()
	logger.SetReportCaller(true)

	return &Logger{
		Logger: logger,
		config: config,
	}
}

// func getOutput(config *config.Config) io.Writer {
// 	switch config.Log.Output {
// 	case "file":
// 		return &lumberjack.Logger{
// 			Filename:   config.Filename,
// 			MaxSize:    config.MaxSize,
// 			MaxAge:     config.MaxAge,
// 			MaxBackups: config.MaxBackups,
// 			Compress:   config.Compress,
// 		}
// 	case "both":
// 		fileWriter := &lumberjack.Logger{
// 			Filename:   config.Filename,
// 			MaxSize:    config.MaxSize,
// 			MaxAge:     config.MaxAge,
// 			MaxBackups: config.MaxBackups,
// 			Compress:   config.Compress,
// 		}
// 		return io.MultiWriter(os.Stdout, fileWriter)
// 	default:
// 		return os.Stdout

// 	}
// }
