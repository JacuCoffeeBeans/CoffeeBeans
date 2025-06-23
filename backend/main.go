// backend/main.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"github.com/rs/cors"
)

// コーヒー豆のデータ構造を定義する
type Bean struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Origin string `json:"origin"`
	Price  int    `json:"price"`
}

// ダミーデータのリスト（仮のデータベース）
var beans = []Bean{
	{ID: 1, Name: "エチオピア イルガチェフェ", Origin: "エチオピア", Price: 1200},
	{ID: 2, Name: "ブラジル サントスNo.2", Origin: "ブラジル", Price: 800},
	{ID: 3, Name: "グアテマラ SHB", Origin: "グアテマラ", Price: 1000},
}

// GET /api/beans のリクエストを処理するハンドラ
func getBeansHandler(w http.ResponseWriter, r *http.Request) {
	// レスポンスの形式がJSONであることをヘッダーで指定
	w.Header().Set("Content-Type", "application/json")

	// beansスライスをJSONに変換してレスポンスとして書き出す
	if err := json.NewEncoder(w).Encode(beans); err != nil {
		// JSONへの変換でエラーが起きたら、サーバーエラーを返す
		http.Error(w, "Failed to encode beans to JSON", http.StatusInternalServerError)
	}
}

func main() {

	mux := http.NewServeMux() // httpルーターをmuxとして定義

	// ルートURLへのアクセス
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Backend server is running!")
	})

	// APIエンドポイントを登録
	mux.HandleFunc("/api/beans", getBeansHandler)

	// ▼▼▼ CORS設定を追加 ▼▼▼
	handler := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173"}, // フロントエンドのURLを許可
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	}).Handler(mux)
	// ▲▲▲ ここまで ▲▲▲

	fmt.Println("Backend server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}