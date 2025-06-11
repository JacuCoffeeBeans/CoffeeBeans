// backend/main.go
package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// ルートURL ("/") へのアクセスがあった場合に、メッセージを返す
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from Go Backend!")
	})

	// サーバー起動のログを出力
	fmt.Println("Backend server is running on http://localhost:8080")
	// 8080ポートでサーバーを起動
	log.Fatal(http.ListenAndServe(":8080", nil))
}