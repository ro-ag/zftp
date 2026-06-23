// SPDX-License-Identifier: Apache-2.0

package zftp

import "gopkg.in/ro-ag/zftp.v2/internal/log"

// LogLevel is a bitmask selecting which categories SetVerbose enables. Its bit
// values mirror internal/log.Level exactly, so SetVerbose's cast is
// value-preserving; combine flags with OR.
type LogLevel log.Level

// NoLog disables every logging category.
const NoLog LogLevel = 0

// The four logging categories are independent, single-bit flags: enabling one
// never implies another. Combine them with OR (or use LogAll).
const (
	LogServer  LogLevel = 1 << iota // 1 — server replies
	LogPassive                      // 2 — passive-mode negotiation
	LogCommand                      // 4 — commands sent
	LogDebug                        // 8 — verbose debug
)

// LogAll enables every logging category.
const LogAll = LogServer | LogPassive | LogCommand | LogDebug
