package zftp

import "gopkg.in/ro-ag/zftp.v1/internal/log"

type logLevel log.Level

const (
	NoLog logLevel = iota << 1
	LogServer
	LogPassive
	LogCommand
	LogDebug
	LogAll = LogServer | LogPassive | LogCommand | LogDebug
)
