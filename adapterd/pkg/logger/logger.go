package logger

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"

	"github.com/Pluslab/cyphonic-adapter/adapterd/config"
	"github.com/Pluslab/cyphonic-adapter/adapterd/domain"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	jsonEncoding    string = "json"
	consoleEncoding string = "console"

	defaultEncoding = consoleEncoding
)

func parseEncodingType(in string) string {
	switch in {
	case "json":
		return jsonEncoding
	case "console":
		return consoleEncoding
	default:
		return defaultEncoding
	}
}

// InitLogger provides logging with custom logger.
func InitLogger(conf *config.Config) {
	log.Printf("LoggerSetting: %+v", conf.Logging)

	filePath, err := getFilePath(conf.Logging)
	if err != nil {
		panic(err)
	}

	logger, err := newZapConfig(conf.Logging, filePath).Build(zap.AddCallerSkip(1))
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := logger.Sync(); err != nil {
			log.Println(err)
		}
	}()

	zap.ReplaceGlobals(logger)
}

func getFilePath(setting *config.Logging) (string, error) {
	var logDir string
	var filePath string

	if len(setting.FileName) > 0 {
		if len(setting.LogDir) > 0 {
			logDir = setting.LogDir
			filePath = fmt.Sprintf("%s/%s", setting.LogDir, setting.FileName)
		} else {
			filePath = setting.FileName
		}
	} else {
		if len(setting.LogDir) > 0 {
			err := errors.New("FileName is empty")
			return "", err
		}
	}

	if len(logDir) > 0 {
		if err := os.MkdirAll(logDir, 0777); err != nil {
			return "", fmt.Errorf("failed to make dir=[%s]: %w", logDir, err)
		}
	}

	return filePath, nil
}

func newZapConfig(setting *config.Logging, filePath string) zap.Config {
	var c zap.Config

	if setting.Output {
		c = zap.Config{
			Level:       zap.NewAtomicLevelAt(zapcore.DebugLevel),
			Development: false,
			Encoding:    parseEncodingType(setting.Encoding),
			Sampling: &zap.SamplingConfig{
				Initial:    100,
				Thereafter: 100,
			},
			EncoderConfig:    newEncoderConfig(),
			OutputPaths:      []string{"stdout"},
			ErrorOutputPaths: []string{"stderr"},
		}

		if len(filePath) > 0 {
			if setting.Rotate != nil && setting.Rotate.MaxSize > 0 {
				c.OutputPaths = []string{"stdout", fmt.Sprintf("lumberjack:%s", filePath)}
				c.OutputPaths = []string{"stderr", fmt.Sprintf("lumberjack:%s", filePath)}

				setLogWriter(setting.Rotate, filePath)
			} else {
				c.OutputPaths = []string{"stdout", filePath}
				c.ErrorOutputPaths = []string{"stderr", filePath}
			}
		}
	} else {
		c = zap.Config{
			Level:       zap.NewAtomicLevelAt(zapcore.DebugLevel),
			Development: false,
			Encoding:    parseEncodingType(setting.Encoding),
			Sampling: &zap.SamplingConfig{
				Initial:    100,
				Thereafter: 100,
			},
			EncoderConfig: newEncoderConfig(),
		}
	}

	return c
}

func newEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:   "eventTime",
		LevelKey:  "level",
		CallerKey: "caller",
		// StacktraceKey: "stacktrace",
		LineEnding:   zapcore.DefaultLineEnding,
		EncodeLevel:  zapcore.CapitalLevelEncoder,
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}
}

type lumberjackSink struct {
	*lumberjack.Logger
}

func (lumberjackSink) Sync() error {
	return nil
}

func setLogWriter(setting *config.LogRotateSetting, filePath string) {
	ll := lumberjack.Logger{
		Filename:   filePath,
		MaxSize:    setting.MaxSize,
		MaxBackups: setting.MaxBackups,
		MaxAge:     setting.MaxAge,
		Compress:   setting.Compress,
	}
	err := zap.RegisterSink("lumberjack", func(*url.URL) (zap.Sink, error) {
		return lumberjackSink{
			Logger: &ll,
		}, nil
	})

	if err != nil {
		panic(err)
	}
}

type ErrCode struct {
	code string
}

func Err(code domain.StatusCode) *ErrCode {
	err := &ErrCode{
		code: strconv.Itoa(int(code)),
	}
	return err
}

// Log level: Debug
func Debug(message ...interface{}) {
	zap.L().Debug("", zap.String("errCode", ""), zap.Any("message", message))
}

func (e *ErrCode) Debug(message interface{}) {
	if e != nil && len(e.code) > 0 {
		zap.L().Debug(message.(string), zap.String("errCode", e.code))
	} else {
		Debug(message)
	}
}

// Log level: Info
func Info(message ...interface{}) {
	zap.L().Info("", zap.String("errCode", ""), zap.Any("message", message))
}

func (e *ErrCode) Info(message interface{}) {
	if e != nil && len(e.code) > 0 {
		zap.L().Info("", zap.String("errCode", e.code), zap.Any("message", message))
	} else {
		Info(message)
	}
}

// Log level: Warning
func Warn(message ...interface{}) {
	zap.L().Warn("", zap.String("errCode", ""), zap.Any("message", message))
}

func (e *ErrCode) Warn(message interface{}) {
	if e != nil && len(e.code) > 0 {
		zap.L().Warn("", zap.String("errCode", e.code), zap.Any("message", message))
	} else {
		Warn(message)
	}
}

// Log level: Error
func Error(message interface{}) {
	zap.L().Error("", zap.String("errCode", ""), zap.Any("message", message))
}

func (e *ErrCode) Error(message interface{}) {
	if e != nil && len(e.code) > 0 {
		zap.L().Error("", zap.String("errCode", e.code), zap.Any("message", message))
	} else {
		Error(message)
	}
}
