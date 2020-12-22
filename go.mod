module github.com/cameronelliott/sfu186j

go 1.15

require (
	github.com/cameronelliott/janus-go v0.0.0-00010101000000-000000000000
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	nhooyr.io/websocket v1.8.6
)

replace github.com/cameronelliott/janus-go => ../janus-go
