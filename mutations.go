// SPDX-License-Identifier: Apache-2.0

package zftp

// Delete removes a file or dataset on the server with a DELE command. name is the
// HFS path or a quoted dataset name ('USER.DATA'). A 550 (not found / not
// permitted) is returned as a *ReturnError; match it with errors.Is(err, CodeError(550)).
func (s *FTPSession) Delete(name string) error {
	_, err := s.SendCommand(CodeFileActionOK, "DELE", name)
	return err
}
