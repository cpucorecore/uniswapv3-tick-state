package main

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

var Logger *zap.Logger

func init() {
	InitLoggerForTest()
}

func InitLoggerForTest() {
	Logger, _ = zap.NewDevelopment()
	return
}

func InitLogger() {
	if !G.Log.Async {
		Logger, _ = zap.NewDevelopment()
		return
	}

	buffer := &zapcore.BufferedWriteSyncer{
		Size:          G.Log.AsyncBufferSizeByByte,
		FlushInterval: time.Second * time.Duration(G.Log.AsyncFlushIntervalBySecond),
		WS:            os.Stdout,
	}
	writeSyncer := zapcore.AddSync(buffer)

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		writeSyncer,
		zapcore.DebugLevel,
	)

	Logger = zap.New(core)
}
