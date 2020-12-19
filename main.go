package main

import (
	//"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	//"github.com/spf13/cobra/cobra/cmd"
	"nhooyr.io/websocket"
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

//var silenceJanus = flag.Bool("silence-janus", false, "if true will throw away janus output")
var debug = flag.Bool("debug", false, "enable debug output")

func killjanus(cmd *exec.Cmd) {
	log.Println("killing janus")
	_ = cmd.Process.Kill()
}

func main() {
	var err error
	flag.Parse()

	var deb *log.Logger = log.New(ioutil.Discard, "", 0)
	if *debug {
		deb = log.New(os.Stdout, "", log.Lmicroseconds)
		deb.Println("debugging enabled")
	}

	cmd := exec.Command("/usr/local/bin/janus")
	//janus doesnt appear to read stdin, and so is immune to closing this: cmd.StdinPipe().Close()   :(

	showJanus := *debug

	// if cmd.Stdout, cmd.Stderr are left as default, janus will exit immediatly!
	if showJanus {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = nil
		cmd.Stderr = os.Stderr // XXX undecided, nil or os.stderr
	}
	log.SetFlags(log.Lmicroseconds)
	log.Println("starting janus")
	err = cmd.Start()
	checkPanic(err)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		_ = cmd.Process.Kill() // not perfect but okay for development testing
	}()
	defer killjanus(cmd) // not perfect but okay for development testing

	//janus waiter thread
	go func() {
		fmt.Printf("Waiting for Janus to finish...\n")
		if err := cmd.Wait(); err != nil {
			fmt.Printf("Janus finished with error: %v\n", err)
			if exiterr, ok := err.(*exec.ExitError); ok {
				// The program has exited with an exit code != 0
				// This works on both Unix and Windows (mac too?). Although package
				// syscall is generally platform dependent, WaitStatus is
				// defined for both Unix and Windows and in both cases has
				// an ExitStatus() method with the same signature.
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

	//connect to janus
	ctx := context.Background()
	time.Sleep(time.Millisecond * 100) // give janus time to start
	c, err := janusConnect(ctx, "ws://localhost:8188/ws")
	log.Println("called connect")
	checkPanic(err)
	//checkFatal(err,"janus dial() failure")

	req := make(map[string]interface{}, 10)
	req["janus"] = "create"
	id := txid
	txid++
	req["transaction"] = strconv.FormatUint(id, 10)
	data, err := json.Marshal(req)
	checkPanic(err)

	fmt.Println(">", string(data))
	err = c.Write(ctx, websocket.MessageText, data)

	for {
		_, m, err := c.Read(ctx)
		if err != nil {
			break
		}
		fmt.Println("<", string(m))
	}
	// err non nil
	// probably closed socket to janus,
	// which means janus crashed or shutdown
	// both are an error
	println("janus read failure", err) // log to stderr
	os.Exit(-1)
}

var txid uint64 = 100

func janusConnect(ctx context.Context, url string) (*websocket.Conn, error) {
	opts := &websocket.DialOptions{Subprotocols: []string{"janus-protocol"}}

	conn, _, err := websocket.Dial(ctx, url, opts)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
