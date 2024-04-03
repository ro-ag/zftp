package zftp

import "gopkg.in/ro-ag/zftp.v1/internal/log"

type LogLevel log.Level

const (
	NoLog LogLevel = iota << 1
	LogServer
	LogPassive
	LogCommand
	LogDebug
	LogAll = LogServer | LogPassive | LogCommand | LogDebug
)
