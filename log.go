package main

import (
	"fmt"
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
}

func InitLogger() {
	if !G.Log.Async {
		Log, _ = zap.NewDevelopment()
		return
	}

	buffer := &zapcore.BufferedWriteSyncer{
		Size:          G.Log.BufferSize,
		FlushInterval: time.Second * time.Duration(G.Log.FlushInterval),
		WS:            os.Stdout,
	}
	writeSyncer := zapcore.AddSync(buffer)

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logLevel, err := zapcore.ParseLevel(G.Log.Level)
	if err != nil {
		panic(fmt.Sprintf("log level error: %v, level:[%s]", err, G.Log.Level))
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		writeSyncer,
		logLevel,
	)

	Log = zap.New(core)
}
