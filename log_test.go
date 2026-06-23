// SPDX-License-Identifier: Apache-2.0

package zftp

import (
	"testing"

	"gopkg.in/ro-ag/zftp.v2/internal/log"
)

// TestLogLevelMirrorsInternal guards the invariant that the public LogLevel
// constants carry the exact bit values of internal/log.Level. SetVerbose maps
// one onto the other with a plain cast (log.SetLevel(log.Level(level))), so any
// drift between the two blocks would silently mis-route logging categories.
func TestLogLevelMirrorsInternal(t *testing.T) {
	cases := []struct {
		name     string
		public   LogLevel
		internal log.Level
	}{
		{"NoLog/None", NoLog, log.None},
		{"LogServer/ServerLevel", LogServer, log.ServerLevel},
		{"LogPassive/PassiveLevel", LogPassive, log.PassiveLevel},
		{"LogCommand/CommandLevel", LogCommand, log.CommandLevel},
		{"LogDebug/DebugLevel", LogDebug, log.DebugLevel},
		{"LogAll/All", LogAll, log.All},
	}

	for _, c := range cases {
		// Values must match...
		if uint32(c.public) != uint32(c.internal) {
			t.Errorf("%s: public=%d internal=%d — values must match", c.name, uint32(c.public), uint32(c.internal))
		}
		// ...and the cast SetVerbose performs must be value-preserving.
		if log.Level(c.public) != c.internal {
			t.Errorf("%s: log.Level(public)=%d != internal=%d", c.name, log.Level(c.public), c.internal)
		}
	}
}
