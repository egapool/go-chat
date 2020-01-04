package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/egapool/go-chat/trace"
	"github.com/joho/godotenv"
	"github.com/stretchr/gomniauth"
	"github.com/stretchr/gomniauth/providers/google"
	"github.com/stretchr/objx"
)

var avatars Avatar = TryAvatars{
	UseFileSystemAvatar,
	UseAuthAvatar,
	UseGravatar,
}

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
	data := map[string]interface{}{
		"Host":        r.Host,
		"ws_protocol": os.Getenv("WS_PROTOCOL"),
	}
	if authCookie, err := r.Cookie("auth"); err == nil {
		data["UserData"] = objx.MustFromBase64(authCookie.Value)
	}
	t.templ.Execute(w, data)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		//log.Fatal("Error loading .env file")
	}

	secretKey := os.Getenv("SECURITY_KEY")
	googleOauthKey := os.Getenv("GOOGLE_OAUTH_KEY")
	googleOauthSecret := os.Getenv("GOOGLE_OAUTH_SECRET")
	gomniauth.SetSecurityKey(secretKey)
	gomniauth.WithProviders(
		google.New(googleOauthKey, googleOauthSecret, os.Getenv("HOST")+"/auth/callback/google"))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r := newRoom()
	r.tracer = trace.New(os.Stdout)
	// func Handle 第2引数に type http.Handler
	// http.Handle("/", &templateHandler{filename: "chat.html"})
	http.Handle("/chat", MustAuth(&templateHandler{filename: "chat.html"}))
	http.Handle("/login", &templateHandler{filename: "login.html"})
	http.HandleFunc("/auth/", loginHandler)
	http.Handle("/room", r)
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:   "auth",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
		w.Header()["Location"] = []string{"/chat"}
		w.WriteHeader(http.StatusTemporaryRedirect)
	})
	http.Handle("/upload", &templateHandler{filename: "upload.html"})
	http.HandleFunc("/uploader", uploaderHandler)
	http.Handle("/avatars/",
		http.StripPrefix("/avatars/",
			http.FileServer(http.Dir("./avatars"))))
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
