package logging

import (
    "os"
    "time"

    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

// InitLogger initializes zap logger according to ENV (production vs development).
// Returns both the raw logger and a sugared logger for convenience.
func InitLogger() (*zap.Logger, *zap.SugaredLogger, error) {
    env := os.Getenv("ENV")
    var logger *zap.Logger
    var err error
    if env == "production" {
        cfg := zap.NewProductionConfig()
        cfg.EncoderConfig.TimeKey = "timestamp"
        cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
        cfg.OutputPaths = []string{"stdout"}
        logger, err = cfg.Build()
    } else {
        // development friendly console logger
        cfg := zap.NewDevelopmentConfig()
        cfg.EncoderConfig.TimeKey = "timestamp"
        cfg.EncoderConfig.EncodeTime = zapcoreISO8601
        logger, err = cfg.Build()
    }
    if err != nil {
        return nil, nil, err
    }
    sugar := logger.Sugar()
    // add a warmup log
    sugar.Infow("logger initialized", "env", env, "time", time.Now())
    return logger, sugar, nil
}

// zapcoreISO8601 is a small helper to avoid importing zap/core directly in templates.
func zapcoreISO8601(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
    enc.AppendString(t.Format(time.RFC3339))
}
