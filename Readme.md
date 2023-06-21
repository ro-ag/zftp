# zFTP

[![Go Reference](https://pkg.go.dev/badge/gopkg.in/ro-ag/zftp.v0.svg)](https://pkg.go.dev/gopkg.in/ro-ag/zftp.v0)

zftp is a Go-based FTP library that provides a high-level interface for interacting with FTP servers. This library is designed to work with IBM mainframe FTP servers and supports specific features such as setting the file type and end-of-line sequences.

## Features

- Opening an FTP session with `Open(server string) (*FTPSession, error)`
- Setting the verbosity of logging with `SetVerbose(v bool)`
- Securing the FTP session using TLS with `AuthTLS(tlsConfig *tls.Config) error`
- Closing an FTP session with `Close() error`
- Logging in with `Login(user, pass string) error`
- Sending a command to the FTP server with `SendCommandWithContext(ctx context.Context, expect ReturnCode, command string, a ...string) (string, error)`
- Getting the system type of the FTP server with `System() string`
- Setting the end-of-line sequence for the FTP server with `SetRetrieveEOL(eol LineBreaker) error`
- Setting the end-of-line wide characters sequence for the FTP server with `SetRetrieveWideCharEOL(eol LineBreaker) error`
- Sending a file to the FTP server with `Put(ctx context.Context, localFile, remoteFile string, options ...PutOption) error`
- Getting a file from the FTP server with `Get(ctx context.Context, remoteFile, localFile string, options ...GetOption) error`
- Listing files in a directory on the FTP server with `List(ctx context.Context, remoteDir string, options ...ListOption) ([]*FTPFile, error)`

## Example Usage

The library is used by creating an `FTPSession` and calling methods on it to interact with the FTP server. Here is a basic example:

```go
package main

import (
	"crypto/tls"
	"gopkg.in/ro-ag/zftp.v0"
	"log"
)

func main() {
	// Open a connection to the FTP server
	session, err := zftp.Open("myftpserver.com:21")
	if err != nil {
		log.Fatal(err)
	}

	// Set up a TLS connection
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	err = session.AuthTLS(tlsConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Log in to the FTP server
	err = session.Login("myusername", "mypassword")
	if err != nil {
		log.Fatal(err)
	}

	// List the files in the root directory
	files, err := session.List("*")
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		log.Println(file)
	}

	// Close the connection to the FTP server
	err = session.Close()
	if err != nil {
		log.Fatal(err)
	}
}
```