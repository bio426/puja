package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type PubBody struct {
	Event string
	Data  string
}

var messageChannels = make(map[chan []byte]bool)

func formatEvent(event string, data string) []byte {
	payload := "event: " + event + "\n"
	datalines := strings.Split(data, "\n")
	for _, line := range datalines {
		payload = payload + "data: " + line + "\n"
	}
	return []byte(payload + "\n")
}

func subscribe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	currentChannel := make(chan []byte)
	messageChannels[currentChannel] = true

	for {
		select {
		case msg := <-currentChannel:
			w.Write(formatEvent("message", string(msg)))
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			delete(messageChannels, currentChannel)
			return
		}
	}
}

func publish(w http.ResponseWriter, r *http.Request) {
	msg := r.FormValue("message")
	log.Print(msg)
	if msg == "" {
		msg = "raaa"
	}
	go func() {
		for channel := range messageChannels {
			channel <- []byte(msg)
		}
	}()
	w.Write([]byte("ok.\n"))
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/pub", publish)
	r.Get("/sub", subscribe)

	ticker := time.NewTicker(time.Second * 5)
	go func() {
		for {
			select {
			case time := <-ticker.C:
				for channel := range messageChannels {
					channel <- []byte(time.String())
				}
			}
		}
	}()
	defer ticker.Stop()

	http.ListenAndServe(":1323", r)
}
