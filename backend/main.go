// backend/main.go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/rs/cors"
)

// newRouter は、このアプリケーションのすべてのルートを含むルーターをセットアップして返す
func newRouter() http.Handler {
	mux := http.NewServeMux() // httpルーターをmuxとして定義

	// ルートURLへのアクセス
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Backend server is running!")
	})
	// APIエンドポイントを登録
	mux.HandleFunc("/api/beans", getBeansHandler)
	mux.HandleFunc("/api/beans/{id}", getBeanHandler)

	// CORS設定
	return cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173"}, // フロントエンドのURLを許可
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	}).Handler(mux)
}

func main() {
	handler := newRouter()
	fmt.Println("Backend server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
