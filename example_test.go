// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"fmt"
	"log"

	zftp "gopkg.in/ro-ag/zftp.v2"
)

// Example shows the basic flow: connect, log in, list datasets, and download a
// member. It is compiled (so the README snippet cannot rot) but not executed,
// since it has no Output comment and would need a live host.
func Example() {
	// Address is host:port.
	s, err := zftp.Open("mainframe.example.com:21")
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	if err := s.Login("USER", "PASSWORD"); err != nil {
		log.Fatal(err)
	}

	datasets, err := s.ListDatasets("USER.*")
	if err != nil {
		log.Fatal(err)
	}
	for _, d := range datasets {
		fmt.Printf("%s recfm=%s lrecl=%s\n", d.Name(), d.Recfm.String(), d.Lrecl.String())
	}

	if err := s.Get("USER.SOURCE(MEMBER)", "member.txt", zftp.TypeBinary); err != nil {
		log.Fatal(err)
	}
}
