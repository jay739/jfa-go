module github.com/jay739/omnifin/scripts/yaml

replace github.com/jay739/omnifin/common => ../../common

replace github.com/jay739/omnifin/logmessages => ../../logmessages

go 1.22.4

require (
	github.com/fatih/color v1.18.0
	github.com/goccy/go-yaml v1.18.0
	github.com/jay739/omnifin/common v0.0.0-20251123201034-b1c578ccf49f
)

require (
	github.com/jay739/omnifin/logmessages v0.0.0-20240806200606-6308db495a0a // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.25.0 // indirect
)
