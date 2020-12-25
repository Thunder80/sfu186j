package main

import (
	"context"

	"fmt"
	"io/ioutil"
	"log"

	"net/http"

	janus "github.com/cameronelliott/janus-go"
)

func watchHandle(h *janus.Handle) {
	// wait for event
	for {
		msg := <-h.Events
		
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
			return // exit go routine on handle detach
		case *janus.EventMsg:
			log.Printf("EventMsg %+v", msg.Plugindata.Data)
		}
	}

}

var ingestAttempted bool = false
var ingestHandle *janus.Handle

func rxwhip(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	ctx := req.Context() // should be 'Done()' if socket closes

	log.Println("got rxwhip request")

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		println(err.Error()) //stderr
		return
	}

	offer := string(body)

	// XXX should we validate the sdp?

	if ingestAttempted {
		http.Error(w, "ingest already attached or attempted/please restart", http.StatusServiceUnavailable)
		return
	}
	ingestAttempted = true

	ingestHandle, err = session.Attach(ctx, "janus.plugin.videoroom")
	if err != nil {
		println("unable to janus videoroom plugin")
		panic(err)
	}

	go watchHandle(ingestHandle)

	answer, code, err := publishJanusVideoRoom(ctx, offer)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		println(err.Error()) //stderr
		return
	}

	log.Println("got Answer from Janus, returning to WHIP client")

	if code == http.StatusAccepted {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(answer))
	} else {
		http.Error(w, answer, code)
	}
}

//body and return code
// body is sdp or errorstring
// code can be:
// 202 Accepted (with sdp)
// 307 Temporary Redirect response code. (no sdp, err?)
// 503 Service Unavailable)

func publishJanusVideoRoom(ctx context.Context, offerSDP string) (string, int, error) {

	var err error
	//get a ingestHandle for videoroom

	//this can only fail when a handle detach fails post hangup
	// we are going to consider thatm'handleable'
	// and thus we will ignore the err, so we do NOT use group.Go()
	// we use traditional go f()

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

	_, err = ingestHandle.Message(ctx, req, nil)
	if err != nil {
		return "", 0, fmt.Errorf("ingestHandle.Message fail %v", err)
	}

	msg, err := ingestHandle.Message(ctx, map[string]interface{}{
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
		log.Printf("no jsep or sdp found from janus: %+v\n", msg)

		if val, ok := msg.Plugindata.Data["error_code"]; ok {
			log.Println("janus error code", val)
		}
		if val, ok := msg.Plugindata.Data["error"]; ok {
			log.Println("janus error ", val)
		}
		return "no jsep or sdp found from janus", http.StatusServiceUnavailable, nil
	}

	log.Println("got the answer SDP back from janus")
	return msg.Jsep["sdp"].(string), http.StatusAccepted, nil
}
