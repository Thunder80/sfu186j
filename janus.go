package main

import (
	"context"

	"fmt"
	"io/ioutil"

	// do not import "log", see comments below

	"net/http"

	janus "github.com/cameronelliott/janus-go"
	"golang.org/x/sync/errgroup"
)



func watchHandle(h *janus.Handle) {
	// wait for event
	for {
		select {
		case msg := <-h.Events:
			switch msg := msg.(type) {
			case *janus.SlowLinkMsg:
				log.Println("SlowLinkMsg type ", h.ID)
			case *janus.MediaMsg:
				log.Println("MediaEvent type", msg.Type, " receiving ", msg.Receiving)
			case *janus.WebRTCUpMsg:
				log.Println("WebRTCUp type ", h.ID)
			case *janus.HangupMsg:
				log.Println("HangupEvent type ", h.ID)
				_, err := h.Detach(context.Background())
				if err != nil {
					println(err.Error())
				}
			case *janus.EventMsg:
				log.Printf("EventMsg %+v", msg.Plugindata.Data)
			}
		}
	}
}

func rxwhip(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	ctx := req.Context() // should be 'Done()' if socket closes

	fmt.Println("got rxwhip request")

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		println(err.Error()) //stderr
		return
	}

	offer := string(body)

	// XXX should we validate the sdp?

	answer, code, err := publishJanusVideoRoom(ctx, offer)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		println(err.Error()) //stderr
		return
	}

	fmt.Println("got Answer from Janus, returning to WHIP client")

	if code == http.StatusAccepted {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(answer))
	} else {
		http.Error(w, answer, code)
	}
}

// XXX if we dont need to run watch handle,
// maybe we do away with all this

// no error, this will outlive the http request and it's context,
// we apparently need to keep the janus session up as long
// as we want the videoroom publisher or subscriber to work
func longLifeJanusSession(session *janus.Session, handle *janus.Handle) {



	ctx := context.Background() //never gets cancelled

	g, ctx := errgroup.WithContext(ctx)

	// XXX
	//g.Go(func() error { return session.KeepAliveSender(ctx) })
	g.Go(func() error { return watchHandle(ctx, handle) })

	err := g.Wait()
	if err != nil {
		println(fmt.Sprintf("session ended with error %v", err))
	}

}

//body and return code
// body is sdp or errorstring
// code can be:
// 202 Accepted (with sdp)
// 307 Temporary Redirect response code. (no sdp, err?)
// 503 Service Unavailable)

func publishJanusVideoRoom(ctx context.Context, offerSDP string) (string, int, error) {

	//no defer session.Destroy()
	// one handle per publisher or subscribera
	handle, err := session.Attach(ctx, "janus.plugin.videoroom")
	if err != nil {
		return "", 0, fmt.Errorf("session.Attach fail %v", err)
	}

	//no defer handle.Detach()

	// WE DONT WANT THIS FUNCTION TO TERMINATE WHEN SOCKET CLOSES
	// so we do not pass a context
	go longLifeJanusSession(session, handle)

	// We only allow a single publisher per room at this time
	// so we keep id==1
	// if we remove id below, multiple publishers per room are allowed
	// maybe allow that as an option

	req := map[string]interface{}{
		"request": "join",
		"ptype":   "publisher",
		"room":    1234,
		"id":      1, // janus only allow a single publisher per id
	}


	_, err = handle.Message(ctx, req, nil)
	if err != nil {
		return "", 0, fmt.Errorf("handle.Message fail %v", err)
	}

	msg, err := handle.Message(ctx, map[string]interface{}{
		"request": "publish",
		"audio":   true,
		"video":   true,
		"data":    false,
		// "audiocodec" : "<audio codec to prefer among the negotiated ones; optional>",
		// "videocodec" : "<video codec to prefer among the negotiated ones; optional>",
		// "bitrate" : <bitrate cap to return via REMB; optional, overrides the global room value if present>,
		// "record" : <true|false, whether this publisher should be recorded or not; optional>,
		// "filename" : "<if recording, the base path/file to use for the recording files; optional>",
		// "display" : "<new display name to use in the room; optional>",
		// "audio_level_average" : "<if provided, overrided the room audio_level_average for this user; optional>",
		// "audio_active_packets" : "<if provided, overrided the room audio_active_packets for this user; optional>"
	}, map[string]interface{}{
		"type":    "offer",
		"sdp":     offerSDP,
		"trickle": false, // trickle false!!!
	})
	if err != nil {
		return "", 0, fmt.Errorf("handle.Message fail %v", err)
	}

	if msg.Jsep == nil || msg.Jsep["sdp"] == nil {
		fmt.Printf("no jsep or sdp found from janus: %+v\n", msg)

		if val, ok := msg.Plugindata.Data["error_code"]; ok {
			fmt.Println("janus error code", val)
		}
		if val, ok := msg.Plugindata.Data["error"]; ok {
			fmt.Println("janus error ", val)
		}
		return "no jsep or sdp found from janus", http.StatusServiceUnavailable, nil	
	}

	fmt.Println("got the answer SDP back from janus")
	return msg.Jsep["sdp"].(string), http.StatusAccepted, nil
}





