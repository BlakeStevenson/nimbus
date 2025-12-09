package logging

import (
	"go.uber.org/zap"
)

// NewLogger creates a new structured logger
func NewLogger(isDevelopment bool) (*zap.Logger, error) {
	var cfg zap.Config

	if isDevelopment {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}

	return cfg.Build()
}

// NewSugaredLogger creates a new sugared logger for easier use
func NewSugaredLogger(isDevelopment bool) (*zap.SugaredLogger, error) {
	logger, err := NewLogger(isDevelopment)
	if err != nil {
		return nil, err
	}
	return logger.Sugar(), nil
}
