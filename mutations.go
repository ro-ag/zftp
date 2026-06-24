// SPDX-License-Identifier: Apache-2.0

package zftp

import "context"

// Delete removes a file or dataset on the server with a DELE command. name is the
// HFS path or a quoted dataset name ('USER.DATA'). A 550 (not found / not
// permitted) is returned as a *ReturnError; match it with errors.Is(err, CodeError(550)).
func (s *FTPSession) Delete(name string) error {
	_, err := s.SendCommand(CodeFileActionOK, "DELE", name)
	return err
}

// Mkdir creates a directory on the server with an MKD command (HFS directory or,
// under SITE DIRECTORYMODE, a dataset qualifier). A 550 is returned as a
// *ReturnError.
func (s *FTPSession) Mkdir(path string) error {
	_, err := s.SendCommand(CodeDirCreated, "MKD", path)
	return err
}

// Rename renames a file or dataset from -> to using the RNFR/RNTO sequence. The
// two round-trips are issued under a single hold of the session mutex so no other
// goroutine's command can interleave between them, keeping *FTPSession safe to
// share across goroutines. A failing RNFR (e.g. 550) is returned without sending
// RNTO.
func (s *FTPSession) Rename(from, to string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.sendLocked(context.Background(), CodeNeedInfo, "RNFR", from); err != nil {
		return err
	}
	_, err := s.sendLocked(context.Background(), CodeFileActionOK, "RNTO", to)
	return err
}
