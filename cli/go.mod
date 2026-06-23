module github.com/ro-ag/zftp/cli

go 1.21

require gopkg.in/ro-ag/zftp.v2 v2.0.0

// During development (before v2.0.0 is tagged) and for local builds, resolve the
// library from the parent directory. Release tooling can drop this.
replace gopkg.in/ro-ag/zftp.v2 => ../
