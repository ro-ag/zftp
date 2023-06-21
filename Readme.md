# zFTP

[![Go Reference](https://pkg.go.dev/badge/gopkg.in/ro-ag/zftp.v0.svg)](https://pkg.go.dev/gopkg.in/ro-ag/zftp.v0)

The zftp package provides a convenient and feature-rich way to work with mainframe FTP servers, specifically designed for z/OS systems. It offers capabilities to interact with z/OS datasets, execute FTP commands tailored for mainframe operations, and handle mainframe-specific attributes and file transfer modes.

## Features

- **z/OS Dataset Support**: Perform operations on z/OS datasets, including retrieving attributes, checking migration status, and verifying dataset size.

- **FTP Commands for Mainframe Operations**: Execute FTP commands specific to z/OS systems, such as listing datasets, retrieving members of a partitioned dataset, and performing file transfers with correct attributes and formats.

- **Working with Mainframe Attributes**: Easily work with mainframe dataset attributes, including volume, unit, RECFM (record format), LRECL (record length), BLKSIZE (block size), and DSORG (dataset organization).

- **Migrated and Not Mounted Dataset Handling**: Identify migrated and not mounted datasets and handle them accordingly in your application logic.

- **Transfer Modes**: Support both ASCII and binary transfer modes required for mainframe file transfers.

- **Verification of File Size**: Verify the transferred file's size to ensure accuracy, particularly important on mainframe systems with gzip format limitations.

- **GetAndGzip**: Retrieve and compress a file in a single step, saving bandwidth and storage space.

## Installation

To install the zftp package, use the following `go get` command:

```bash
go get gopkg.in/ro-ag/zftp.v0
```

## Usage

Here are some of the most important functions provided by the zftp package:

- `Open(hostname string) (*FTPSession, error)`: Open an FTP session to the specified hostname, returning a session instance for further operations.

- `(*FTPSession) Login(username, password string) error`: Log in to the FTP server using the provided username and password.

- `(*FTPSession) Get(remoteFile, localFile string, transferType TransferType) error`: Retrieve a file from the FTP server and store it locally, specifying the transfer type as ASCII or binary.

- `(*FTPSession) Put(localFile, remoteFile string, transferType TransferType) error`: Upload a local file to the FTP server, specifying the transfer type as ASCII or binary.

- `(*FTPSession) List(remotePath string) ([]string, error)`: List files and directories at the specified remote path on the FTP server.

- `(*FTPSession) ListDatasets(remotePath string) ([]hfs.Dataset, error)`: List z/OS datasets at the specified remote path, including dataset attributes.

- `(*FTPSession) GetAndGzip(remoteFile, localFile string, transferType TransferType) error`: Retrieve a file from the FTP server, compress it using gzip format, and store it locally in a single step.

Refer to the [GoDoc](https://pkg.go.dev/gopkg.in/ro-ag/zftp.v0) for detailed documentation and more functions provided by the package.

## Example

Here's an example that demonstrates the basic usage of the zftp package:

```go
package main

import (
	"fmt"
	"gopkg.in/ro-ag/zftp.v0"
)

func main() {
	// Open an FTP session to the mainframe server
	s, err := zftp.Open("example.com")
	if err != nil {
		fmt.Println("Failed to open FTP session:", err)
		return
	}

	// Log in to the FTP server
	err = s.Login("username", "password")
	if err != nil {
		fmt.Println("Failed to log in:", err)
		return
	}

	// Retrieve a file from the FTP server and store it locally
	err = s.Get("remote_file.txt", "local_file.txt", zftp.TypeAscii)
	if err != nil {
		fmt.Println("Failed to retrieve file:", err)
		return
	}

	// Retrieve and compress a file from the FTP server using gzip format
	err = s.GetAndGzip("remote_file.txt", "local_file.txt.gz", zftp.TypeAscii)
	if err != nil {
		fmt.Println("Failed to retrieve and compress file:", err)
		return
	}

	// Close the FTP session
	err = s.Close()
	if err != nil {
		fmt.Println("Failed to close FTP session:", err)
		return
	}

	fmt.Println("File retrieved and compressed successfully!")
}
```

The `GetAndGzip` function allows you to retrieve a file from the FTP server and compress it using gzip format in a single step. This can be beneficial in scenarios where you need to conserve bandwidth and storage space. By compressing the file during the retrieval process, you reduce the size of the transferred data, resulting in faster downloads and reduced storage requirements. The `GetAndGzip` function simplifies this process by handling the retrieval and compression in a single function call, streamlining your file transfer operations.

## Contributing

Contributions to the zftp package are welcome! Please open an issue to discuss any proposed changes or improvements.

## License

The zftp package is licensed under the MIT License. See the [LICENSE](./LICENSE) file for more information.

---
