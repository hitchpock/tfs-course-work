package log

import (
	"go.uber.org/zap"
)

type SugarLogger struct {
	Sugar *zap.SugaredLogger
}

func NewSugarLogger() *SugarLogger {
	sugar := zap.NewExample().Sugar()
	return &SugarLogger{Sugar: sugar}
}

func (s *SugarLogger) Info(args ...interface{}) {
	s.Sugar.Info(args)
}

func (s *SugarLogger) Infof(template string, args ...interface{}) {
	s.Sugar.Infof(template, args)
}

func (s *SugarLogger) Infow(msg string, keysAndValues ...interface{}) {
	s.Sugar.Infow(msg, keysAndValues...)
}

func (s *SugarLogger) Warn(args ...interface{}) {
	s.Sugar.Warn(args)
}

func (s *SugarLogger) Warnf(template string, args ...interface{}) {
	s.Sugar.Warnf(template, args)
}

func (s *SugarLogger) Warnw(msg string, keysAndValues ...interface{}) {
	s.Sugar.Warnw(msg, keysAndValues...)
}

func (s *SugarLogger) Fatal(args ...interface{}) {
	s.Sugar.Fatal(args)
}

func (s *SugarLogger) Fatalf(template string, args ...interface{}) {
	s.Sugar.Fatalf(template, args)
}
