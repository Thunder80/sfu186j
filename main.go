package main

import (
	//"bytes"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"

	janus "github.com/cameronelliott/janus-go"
)

func checkPanic(err error) {
	if err != nil {
		panic(err)
	}
}

// func checkFatal(err error, msg string) {
// 	if err != nil {
// 		println(msg, err) // print to stderr
// 		os.Exit(-1)
// 	}
// }


func logJson(m []byte, wasrx bool) {
	prefix := "<"
	if !wasrx {
		prefix = ">"
	}
	// log message being sent
	var log bytes.Buffer
	_ = json.Indent(&log, m, prefix, "   ")
	log.Write([]byte("\n"))
	_, _ = log.WriteTo(os.Stdout)
}

//logging philosophy
// debugging only goes to io.stdout
// info goes to stderr
// errors go to stderr

func startJanusInstance() {
	cmd := exec.Command("/usr/local/bin/janus")

	showJanus := *debug

	if showJanus {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = nil
		cmd.Stderr = os.Stderr
	}

	err := cmd.Start()
	checkPanic(err)

	go func() {
		fmt.Printf("xWaiting for Janus to finish...\n")
		if err := cmd.Wait(); err != nil {
			fmt.Printf("Janus finished with error: %v\n", err)
			if exiterr, ok := err.(*exec.ExitError); ok {

				if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					fmt.Printf("Janus Exit Status: %d\n", status.ExitStatus())

					os.Exit(status.ExitStatus())
				}
			} else {
				panic("fail deref err")
			}
		} else {
			fmt.Printf("Janus finished without errors\n")
		}

		os.Exit(-1)
	}()
}

var gateway *janus.Gateway
var session *janus.Session

//var silenceJanus = flag.Bool("silence-janus", false, "if true will throw away janus output")
var debug = flag.Bool("debug", false, "enable debug output")
var noStartJanus = flag.Bool("dev-no-janus", false, "developer flag: do not start janus")

func main() { os.Exit(main2()) }
//can combine return X with defer func()
func main2() (exitcode int) {




	var err error
	flag.Parse()

	if *debug {
		log.SetFlags(log.Lmicroseconds | log.LUTC)
		log.Println("debug output IS enabled")
	} else {
		log.Println("debug output NOT enabled")
		log.SetOutput(ioutil.Discard)
		log.SetFlags(0)
	}

	if *noStartJanus {
		log.Println("NOT starting janus")
	} else {
		log.Println("starting janus")
		startJanusInstance()
		time.Sleep(time.Millisecond * 100) 
	}



	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // good practice

	
	janusURL:="ws://localhost:8188/ws"

	gateway, err = janus.Connect(ctx, janusURL)
	if err != nil {
		panic(err)
	}
	fmt.Println("succesfully connected to Janus")

	// not sure:
	// one session per handle, or one per connection
	session, err = gateway.Create(ctx)
	if err != nil {
		panic(err)
	}

	// 60 second timeout on sessions
	// https://janus.conf.meetecho.com/docs/rest.html#WS
	go func() {
		err = session.KeepAliveSender(ctx)
		println("KeepAliveSender died")
		if err != nil {
			panic(err)
		}
	}()

	rxpath:="/rx"
	txpath:="/tx"
	_=txpath

	mux := http.NewServeMux()
	mux.HandleFunc(rxpath, rxwhip)
	//mux.HandleFunc(txpath, rxwhip)
	//mux.HandleFunc("/", hello)

	ln, err := net.Listen("tcp", ":8080")
	checkPanic(err)

	fmt.Println("WHIP ingress listener at:", ln.Addr().String(), rxpath)

	//go killme(ctx)

	err = http.Serve(ln, mux)
	if err != nil {
		panic(err)
	}

	//http.ListenAndServe()

	// err = HTTPS([]string{"kego.com"}, mux) // https automagic
	// panic(err)

	return
}



// func validateMsgtype(jmsg *JanusMessage) (bool, error) {
// 	switch m := jmsg.specific.(type) {
// 	case *ErrorMsg:
// 	case *SuccessMsg:
// 	case *DetachedMsg:
// 	case *InfoMsg:
// 	case *AckMsg:
// 	case *EventMsg:
// 	case *WebRTCUpMsg:
// 	case *MediaMsg:
// 	case *HangupMsg:
// 	case *SlowLinkMsg:
// 	case *TimeoutMsg:
// 	case *PluginData:
// 	default:
// 		z := fmt.Sprintf("unexpected type from janusdecode %+v!", m)
// 		return true, fmt.Errorf(z)
// 	}
// 	return false, nil
// }

// important, janus sessions get closed after 60 secs of inactivity:
//   (search: 60)
// this is needed to keep session alive
// if the session dies, we see stuff to start to shut down.
// run this to keep session open, otherwise, call starts shutdown after 60
