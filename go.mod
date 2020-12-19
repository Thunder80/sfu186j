module github.com/cameronelliott/sfu186j

go 1.15

require (
	github.com/cameronelliott/janus-go v0.0.0-00010101000000-000000000000
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/spf13/cobra v1.1.1
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	nhooyr.io/websocket v1.8.6
)

replace github.com/cameronelliott/janus-go => ../janus-go
