package main

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

var Log *zap.Logger

func init() {
	InitLoggerForTest()
}

func InitLoggerForTest() {
	Log, _ = zap.NewDevelopment()
	return
}

func InitLogger() {
	if !G.Log.Async {
		Log, _ = zap.NewDevelopment()
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

	Log = zap.New(core)
}
