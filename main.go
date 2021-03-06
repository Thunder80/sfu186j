package main

import (
	"context"
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

	"golang.org/x/sync/errgroup"

	janus "github.com/cameronelliott/janus-go"
)

func checkPanic(err error) {
	if err != nil {
		panic(err)
	}
}





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
		log.Printf("xWaiting for Janus to finish...\n")
		if err := cmd.Wait(); err != nil {
			log.Printf("Janus finished with error: %v\n", err)
			if exiterr, ok := err.(*exec.ExitError); ok {

				if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					log.Printf("Janus Exit Status: %d\n", status.ExitStatus())

					os.Exit(status.ExitStatus())
				}
			} else {
				panic("fail deref err")
			}
		} else {
			log.Printf("Janus finished without errors\n")
		}

		os.Exit(-1)
	}()
}

var gateway *janus.Gateway
var session *janus.Session // just a single handle for the program, all handles

//var silenceJanus = flag.Bool("silence-janus", false, "if true will throw away janus output")
var debug = flag.Bool("debug", true, "enable debug output")
var nohtml = flag.Bool("no-html", false, "do not serve any html files, only do WHIP")
var noStartJanus = flag.Bool("dev-no-janus", false, "developer flag: do not start janus")

var info = log.New(os.Stderr, "", log.Lmicroseconds|log.LUTC)

func main() { os.Exit(main2()) }
//can combine return X with defer func()
func main2() (exitcode int) {
	var err error
	flag.Parse()


	if *debug {
		log.SetFlags(log.Lmicroseconds | log.LUTC)
		info.Println("debug output IS enabled")
	} else {
		info.Println("debug output NOT enabled")
		log.SetOutput(ioutil.Discard)
		log.SetFlags(0)
	}

	if *noStartJanus {
		info.Println("NOT starting janus")
	} else {
		info.Println("starting janus")
		startJanusInstance()
		time.Sleep(time.Millisecond * 100)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // good practice

	janusURL := "ws://localhost:8188/ws"

	var group *errgroup.Group
	gateway, group, err = janus.Connect(ctx, janusURL)
	if err != nil {
		panic(err)
	}
	log.Println("succesfully connected to Janus")

	// not sure:
	// one session per handle, or one per connection
	session, err = gateway.Create(ctx)
	if err != nil {
		println("unable to communicate with janus")
		panic(err)
	}

	// 60 second timeout on sessions
	// https://janus.conf.meetecho.com/docs/rest.html#WS
	// This is a fancy 'Go' that allows catching errors
	group.Go(func() error { return session.KeepAliveSender(ctx) })

	rxpath := "/rx"
	txpath := "/tx"
	_ = txpath

	mux := http.NewServeMux()
	mux.HandleFunc(rxpath, rxwhip)

	if !*nohtml {
		buf,err:=ioutil.ReadFile("html/index.html")
	checkPanic(err)

		// handle `/` route
		mux.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
			res.Header().Set("Content-Type", "text/html")
			fmt.Fprint(res, string(buf))
		})
	}
	//mux.HandleFunc(txpath, rxwhip)
	//mux.HandleFunc("/", hello)

	ln, err := net.Listen("tcp", ":8080")
	checkPanic(err)

	log.Println("WHIP ingress listener at:", ln.Addr().String(), rxpath)

	//go killme(ctx)

	group.Go(func() error { return http.Serve(ln, mux) })

	//this should have four goroutines in it
	// 1. connection janus.ping()
	// 2. connection janus.recv()
	// 3. session keepalive sender
	// 4. the http.Serve
	// if any return an error, this will return with it
	err = group.Wait()
	panic(err)
	//http.ListenAndServe()

	// err = HTTPS([]string{"kego.com"}, mux) // https automagic
	// panic(err)

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
