// SPDX-License-Identifier: Apache-2.0

package mockzos

import "strings"

// Script registers raw control reply line(s) for a command. The match key may be
// a full command line ("XSTA (BLOCKSIze") or just a verb ("STAT"); a full-line
// script wins over a verb script over the built-in default.
//
// Reply lines are written verbatim, so callers control the exact return codes and
// multiline continuation form, e.g.:
//
//	srv.Script("STAT", "211-begin", "211 end")
//	srv.Script("XSTA (BLOCKSIze", "211-Record format FB, Lrecl: 80, Blocksize: 27920", "211 *** end of status ***")
func (s *Server) Script(command string, replies ...string) {
	key := strings.ToUpper(strings.TrimSpace(command))
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.ContainsRune(key, ' ') {
		s.lineScripts[key] = replies
	} else {
		s.verbScripts[key] = replies
	}
}

// DataFor registers the payload streamed over the data connection for a download
// command (LIST/NLST/RETR). Pass an empty arg to match any argument for the verb.
//
//	srv.DataFor("LIST", "USER.*", listing)
//	srv.DataFor("RETR", "MY.FILE", contents)
func (s *Server) DataFor(verb, arg, payload string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	verb = strings.ToUpper(strings.TrimSpace(verb))
	if arg == "" {
		s.dataByVerb[verb] = payload
		return
	}
	s.dataByLine[verb+" "+strings.TrimSpace(arg)] = payload
}

// Stored returns the bytes captured for a prior STOR of the given remote name.
func (s *Server) Stored(arg string) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.stored[strings.ToUpper(strings.TrimSpace(arg))]
	return b, ok
}

func (s *Server) scriptFor(line, verb string) ([]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.lineScripts[strings.ToUpper(strings.TrimSpace(line))]; ok {
		return r, true
	}
	if r, ok := s.verbScripts[verb]; ok {
		return r, true
	}
	return nil, false
}

func (s *Server) dataFor(line, verb string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.dataByLine[strings.ToUpper(strings.TrimSpace(line))]; ok {
		return p, true
	}
	if p, ok := s.dataByVerb[verb]; ok {
		return p, true
	}
	return "", false
}
