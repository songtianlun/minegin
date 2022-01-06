package logger

import (
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"minepin-backend/config"
	"os"
)

var Logger *zap.Logger             // 在每一微秒和每一次内存分配都很重要的上下文中，使用Logger，只支持强类型的结构化日志记录
var SugarLogger *zap.SugaredLogger // 在性能很好但不是很关键的上下文中，使用SugaredLogger，支持结构化和 printf 风格

var logLevel zapcore.LevelEnabler

func InitLogger() {
	switch config.GetString("log.logger_level") {
	case "debug":
		logLevel = zap.DebugLevel
	case "info":
		logLevel = zap.InfoLevel
	case "warn":
		logLevel = zap.WarnLevel
	case "error":
		logLevel = zap.ErrorLevel
	case "panic":
		logLevel = zap.PanicLevel
	case "fatal":
		logLevel = zap.FatalLevel
	default:
		logLevel = zap.InfoLevel
	}
	core := zapcore.NewTee(getAllCores()...)

	// AddCaller - 调用函数信息记录到日志中的功能
	// AddCallerSkip - 向上跳 1 层，输出调用封装日志函数的位置
	Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	SugarLogger = Logger.Sugar()
	defer Logger.Sync() // flushes buffer, if any
	defer SugarLogger.Sync()
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder   // 格式化时间显示
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder // 使用大写字母记录日志级别
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getLogWriter() zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   config.GetString("log.logger_file_name"),
		MaxSize:    config.GetInt("log.logger_max_size_mb"),
		MaxBackups: config.GetInt("log.logger_max_file_num"),
		MaxAge:     config.GetInt("log.logger_max_file_day"),
		Compress:   config.GetBool("log.logger_compress"),
	}
	return zapcore.AddSync(lumberJackLogger)
}

func getAllCores() []zapcore.Core {
	var allCore []zapcore.Core
	writeSyncer := getLogWriter()
	encoder := getEncoder()
	// debug 模式或指定输出到屏终端时将日志同时输出到 stdout
	if config.Get(config.MINEPIN_RUNMODE) == "debug" ||
		config.GetBool("log.logger_stdout") {
		consoleDebugging := zapcore.Lock(os.Stdout)
		allCore = append(allCore, zapcore.NewCore(encoder, consoleDebugging, logLevel))
	}
	allCore = append(allCore, zapcore.NewCore(encoder, writeSyncer, logLevel))
	return allCore
}

func DebugF(format string, v ...interface{}) { SugarLogger.Debugf(format, v...) }
func InfoF(format string, v ...interface{})  { SugarLogger.Infof(format, v...) }
func WarnF(format string, v ...interface{})  { SugarLogger.Warnf(format, v...) }
func ErrorF(format string, v ...interface{}) { SugarLogger.Errorf(format, v...) }
func PanicF(format string, v ...interface{}) { SugarLogger.Panicf(format, v...) }
func FatalF(format string, v ...interface{}) { SugarLogger.Fatalf(format, v...) }
func Debug(v ...interface{})                 { SugarLogger.Debug(v...) }
func Info(v ...interface{})                  { SugarLogger.Info(v...) }
func Warn(v ...interface{})                  { SugarLogger.Warn(v...) }
func Error(v ...interface{})                 { SugarLogger.Error(v...) }
func Panic(v ...interface{})                 { SugarLogger.Panic(v...) }
func Fatal(v ...interface{})                 { SugarLogger.Fatal(v...) }
