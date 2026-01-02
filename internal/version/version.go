package version

// Version is the application version, set at build time via -ldflags
var Version = "dev"

// GitCommit is the git commit hash, set at build time via -ldflags
var GitCommit = "unknown"
