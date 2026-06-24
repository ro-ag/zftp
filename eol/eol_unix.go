// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package eol

// System is the host platform's native end-of-line sequence. On non-Windows
// platforms it is [Lf].
const System = Lf
