package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/egapool/go-chat2/trace"
	"github.com/joho/godotenv"
)

// templは1つのテンプレートを表します
type templateHandler struct {
	once     sync.Once
	filename string
	templ    *template.Template
}

func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.once.Do(func() {
		t.templ =
			template.Must(template.ParseFiles(filepath.Join("template",
				t.filename)))
	})
	t.templ.Execute(w, r)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r := newRoom()
	r.tracer = trace.New(os.Stdout)
	// func Handle 第2引数に type http.Handler
	http.Handle("/", &templateHandler{filename: "chat.html"})
	http.Handle("/room", r)

	/*
		チャッ ト関連の処理はバックグラウンドで行われます。
		その結果、メインのスレッドで Web サーバーを実 行できるようになります。
	*/
	go r.run()
	// Webサーバーを開始します
	log.Println("Web Server Stated. http://localhost:" + port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
