module github.com/ro-ag/zftp/cli

go 1.26

require gopkg.in/ro-ag/zftp.v2 v2.0.0

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/term v0.44.0 // indirect
)

// During development (before v2.0.0 is tagged) and for local builds, resolve the
// library from the parent directory. Release tooling can drop this.
replace gopkg.in/ro-ag/zftp.v2 => ../
