package main

import (
	"log"
	"net/http"

	"github.com/egapool/go-chat2/trace"
	"github.com/gorilla/websocket"
)

type room struct {
	// forwardは他のクライアントに転送するためのメッセージを保持するチャネルです。
	forward chan []byte

	/*
		ここでは2つのチャネルと1つのマップが追加されています。
		joinとleaveのチャネルはそれぞれ、
		マップ clients に対するクライアントの追加と削除に使われます。
		チャネルを使わずにこのマッ プを直接操作することは望ましくありません。
		複数の goroutine がマップを同時に変更する可能性が
		生じ、メモリの破壊やその他の予期せぬ状態がもたらされるからです。
	*/
	join    chan *client
	leave   chan *client
	clients map[*client]bool
	tracer  trace.Tracer
}

func newRoom() *room {
	return &room{
		forward: make(chan []byte),
		join:    make(chan *client),
		leave:   make(chan *client),
		clients: make(map[*client]bool),
		tracer:  trace.Off(),
	}
}

func (r *room) run() {
	/*
		いずれかのチャネルにメッセージが届くと、
		select 文の中でそれぞれに対応する case 節が実行されます。
		この case 節のコードは、同時に実行されることはありません。
		この 性質のおかげで、マップ r.clients への変更が
		同時に発生するということが防がれています。
	*/
	for {
		select {
		// join チャネルから取り出している
		case client := <-r.join:
			r.clients[client] = true
			r.tracer.Trace("新しいクライアントが参加しました")
		case client := <-r.leave:
			delete(r.clients, client)
			close(client.send)
			r.tracer.Trace("クライアントが退室しました")
		case msg := <-r.forward:
			r.tracer.Trace("メッセージを受信しました: ", string(msg))
			for client := range r.clients {
				select {
				case client.send <- msg:
					// send message
					r.tracer.Trace("-- クライアントに送信されました")
				default:
					delete(r.clients, client)
					close(client.send)
					r.tracer.Trace(" -- 送信に失敗しました。クライアントをクリーンアップします")
				}
			}
		}
	}
}

const (
	socketBufferSize  = 1024
	messageBufferSize = 256
)

var upgrader = &websocket.Upgrader{
	ReadBufferSize:  socketBufferSize,
	WriteBufferSize: socketBufferSize,
}

func (r *room) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	socket, err := upgrader.Upgrade(w, req, nil)
	if nil != err {
		log.Fatal("ServeHTTP:", err)
		return
	}
	client := &client{
		socket: socket,
		send:   make(chan []byte, messageBufferSize),
		room:   r,
	}
	r.join <- client
	defer func() { r.leave <- client }()
	go client.read()
	client.write()
}
