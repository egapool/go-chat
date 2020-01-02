package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
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
	t.templ.Execute(w, nil)
}

func main() {
	port := os.Getenv("PORT")
	r := newRoom()
	// func Handle 第2引数に type http.Handler
	http.Handle("/", &templateHandler{filename: "chat.html"})
	http.Handle("/room", r)

	/*
		チャッ ト関連の処理はバックグラウンドで行われます。
		その結果、メインのスレッドで Web サーバーを実 行できるようになります。
	*/
	go r.run()
	// Webサーバーを開始します
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
