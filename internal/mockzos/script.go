// SPDX-License-Identifier: Apache-2.0

package mockzos

import (
	"crypto/tls"
	"strings"
)

// EnableTLS makes the server accept AUTH TLS and upgrade the control connection
// to a TLS server session using cfg (which must carry a certificate). It lets
// tests exercise the client's AuthTLS/PBSZ/PROT path over a real, in-process TLS
// handshake. Call it before the client dials.
func (s *Server) EnableTLS(cfg *tls.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tlsConfig = cfg
}

// CompletionReply overrides the closing reply the server sends after a transfer
// completes for the given verb (RETR/LIST/NLST or STOR/STOU/APPE). The default is
// a single "250 transfer completed successfully" line; pass replies to script a
// different terminal code (e.g. "226 ...") or a multiline z/OS JES message that
// carries a job id. The key is a verb.
//
//	srv.CompletionReply("STOR", "250-IT IS KNOWN TO JES AS JOB12345", "250 SUBMIT successful")
//	srv.CompletionReply("RETR", "226 Closing data connection; transfer complete")
func (s *Server) CompletionReply(verb string, replies ...string) {
	key := strings.ToUpper(strings.TrimSpace(verb))
	s.mu.Lock()
	defer s.mu.Unlock()
	s.completionReply[key] = replies
}

// completionReplyFor returns the scripted closing reply for a transfer verb, or
// the default "250 transfer completed successfully" when none is set.
func (s *Server) completionReplyFor(verb string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.completionReply[verb]; ok {
		return r
	}
	return []string{"250 transfer completed successfully"}
}

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

// Withhold makes the server consume the matching command without ever sending a
// reply, modeling a hung or unresponsive control connection. The match key may be
// a full command line ("XSTA (BLOCKSIze") or just a verb ("STAT"), like Script.
// It lets tests drive the client's context-timeout / cancellation paths.
//
//	srv.Withhold("STAT") // the next STAT receives no reply
func (s *Server) Withhold(command string) {
	key := strings.ToUpper(strings.TrimSpace(command))
	s.mu.Lock()
	defer s.mu.Unlock()
	s.withheld[key] = true
}

func (s *Server) isWithheld(line, verb string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.withheld[strings.ToUpper(strings.TrimSpace(line))] {
		return true
	}
	return s.withheld[verb]
}

// Hangup makes the server drop the control connection (without replying) when it
// receives the matching command, modeling a peer that closes the control stream
// — the client's reply read then sees EOF. The match key may be a full command
// line or just a verb, like Script.
//
//	srv.Hangup("STAT") // the next STAT closes the control connection
func (s *Server) Hangup(command string) {
	key := strings.ToUpper(strings.TrimSpace(command))
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hangup[key] = true
}

func (s *Server) isHangup(line, verb string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hangup[strings.ToUpper(strings.TrimSpace(line))] {
		return true
	}
	return s.hangup[verb]
}

// DropControlAfterData makes a download (LIST/NLST/RETR) deliver its data and
// then drop the control connection instead of sending the closing 250 reply, so
// the client's post-transfer reply read hits EOF. The key is a verb.
//
//	srv.DropControlAfterData("LIST")
func (s *Server) DropControlAfterData(verb string) {
	key := strings.ToUpper(strings.TrimSpace(verb))
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dropAfterData[key] = true
}

func (s *Server) isDropAfterData(verb string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dropAfterData[verb]
}

// TruncateData makes a download (LIST/NLST/RETR) abort its data connection with a
// TCP RST (SO_LINGER 0) instead of a clean FIN after sending the payload, while
// the control connection still reports 250. It models a failed/aborted z/OS
// transfer, so the client must fail the operation on the data-stream error even
// though the control reply looks successful. The key is a verb.
func (s *Server) TruncateData(verb string) {
	key := strings.ToUpper(strings.TrimSpace(verb))
	s.mu.Lock()
	defer s.mu.Unlock()
	s.truncate[key] = true
}

func (s *Server) isTruncateData(verb string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.truncate[verb]
}

// HangData makes a download hold its data connection open after sending the
// payload (blocking until the client closes it), so the client's scan stalls and
// a concurrent Close can be exercised. The key is a verb.
func (s *Server) HangData(verb string) {
	key := strings.ToUpper(strings.TrimSpace(verb))
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hangData[key] = true
}

func (s *Server) isHangData(verb string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hangData[verb]
}

// WithholdReplyAfterData makes a download deliver its data and cleanly close the
// data connection, but never send the closing 250 reply, leaving the control
// connection open so the client's post-transfer reply read blocks (and must time
// out). The key is a verb.
func (s *Server) WithholdReplyAfterData(verb string) {
	key := strings.ToUpper(strings.TrimSpace(verb))
	s.mu.Lock()
	defer s.mu.Unlock()
	s.withholdReply[key] = true
}

func (s *Server) isWithholdReplyAfterData(verb string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.withholdReply[verb]
}

// Commands returns a copy of every command line the server has received, in
// order, so tests can assert on the exact control sequence (e.g. TYPE/SITE/REST).
func (s *Server) Commands() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.received))
	copy(out, s.received)
	return out
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
