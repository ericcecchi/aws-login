package awslogin

import "runtime/debug"

// version is set at build time via -ldflags. For binaries installed with
// "go install", it falls back to the module version embedded by the Go
// toolchain. The sentinel "dev" is used for local untagged builds.
var version = "dev"

func init() {
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok &&
			info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
	}
}

const (
	awsSSOCacheDir = "~/.aws/sso/cache"
)
