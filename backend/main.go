// backend/main.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os" // .envファイルを読み込むために追加

	"github.com/rs/cors"
	"github.com/supabase-community/postgrest-go" // Supabaseクライアントを追加
)

// Api構造体にStoreを持たせる準備
// type Api struct {
// 	store *Store
// }

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
	// .envからSupabaseの情報を取得
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_SERVICE_KEY")
	if supabaseURL == "" || supabaseKey == "" {
		log.Fatal("環境変数 SUPABASE_URL と SUPABASE_SERVICE_KEY を設定してください")
	}

	// SupabaseクライアントとStoreを初期化
	headers := map[string]string{
		"apikey":        supabaseKey,
		"Authorization": "Bearer " + supabaseKey,
	}
	client := postgrest.NewClient(supabaseURL+"/rest/v1", "", headers)
	_ = NewStore(client) // store変数はまだ使わないので、`_`で受け取る
	//store := NewStore(client)

	log.Println("Successfully initialized Supabase client!") // 接続準備ができたことをログに出力

	// Apiインスタンスを作成（今後のための準備）
	// api := &Api{store: store}

	handler := newRouter()
	fmt.Println("Backend server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
