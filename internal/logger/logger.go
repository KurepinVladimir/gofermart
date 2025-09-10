package logger

import "go.uber.org/zap"

var Log *zap.Logger

func Init() error {
	l, err := zap.NewDevelopment() // для локальной разработки
	if err != nil {
		return err
	}
	Log = l
	return nil
}

func Close() {
	if Log != nil {
		_ = Log.Sync()
	}
}
