module github.com/jay739/omnifin/scripts/ini

replace github.com/jay739/omnifin/common => ../../common

replace github.com/jay739/omnifin/logmessages => ../../logmessages

go 1.22.4

require (
	github.com/goccy/go-yaml v1.18.0
	github.com/jay739/omnifin/common v0.0.0-00010101000000-000000000000
	gopkg.in/ini.v1 v1.67.0
)

require (
	github.com/jay739/omnifin/logmessages v0.0.0-20240806200606-6308db495a0a // indirect
	github.com/stretchr/testify v1.11.1 // indirect
)
