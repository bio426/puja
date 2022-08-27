package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/net/websocket"
)

type alert struct {
	event   string
	message string
}

var messageChannels = make(map[chan alert]bool)

func formatEvent(event string, data string) []byte {
	payload := strings.Builder{}
	payload.WriteString("id: asd\n")
	payload.WriteString("event: " + event + "\n")
	datalines := strings.Split(data, "\n")
	for _, line := range datalines {
		payload.WriteString("data: " + line + "\n")
	}
	payload.WriteString("\n")

	return []byte(payload.String())
}

func subscribe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	currentChannel := make(chan alert)
	messageChannels[currentChannel] = true

	for {
		select {
		case incomeAlert := <-currentChannel:
			w.Write(formatEvent(incomeAlert.event, incomeAlert.message))
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			delete(messageChannels, currentChannel)
			return
		}
	}
}

func publish(w http.ResponseWriter, r *http.Request) {
	//w.Header().Set("Access-Control-Allow-Origin", "*")
	var body struct {
		Event   string `json:"event"`
		Message string `json:"message"`
	}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&body)
	if err != nil {
		panic(err)
	}
	if body.Message == "" {
		body.Message = "no message"
	}

	go func() {
		for channel := range messageChannels {
			myAlert := alert{event: body.Event, message: body.Message}
			channel <- myAlert
		}
	}()
	w.Write([]byte("ok.\n"))
}

func sockets(w http.ResponseWriter, r *http.Request) {
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		for {
			err := websocket.Message.Send(ws, "Hola manito")
			if err != nil {
				log.Panicf(err.Error())
			}
			msg := ""
			err = websocket.Message.Receive(ws, &msg)
			if err != nil {
				log.Panic(err)
			}
			log.Printf("%s\n", msg)
		}
	}).ServeHTTP(w, r)
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Post("/pub", publish)
	r.Options("/pub", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Vary", "Access-Control-Request-Method")
		w.Header().Set("Vary", "Access-Control-Request-Headers")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Origin, Accept, token")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST,OPTIONS")

		w.WriteHeader(200)
	})
	r.Get("/sub", subscribe)
	r.Get("/ws", sockets)

	ticker := time.NewTicker(time.Second * 5)
	go func() {
		for {
			select {
			case time := <-ticker.C:
				for channel := range messageChannels {
					timeAlert := alert{event: "time", message: time.String()}
					channel <- timeAlert
				}
			}
		}
	}()
	defer ticker.Stop()

	http.ListenAndServe(":1323", r)
}
