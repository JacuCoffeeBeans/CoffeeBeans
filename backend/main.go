// backend/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os" // .envファイルを読み込むために追加

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/cors"
)

// Api はStore（DB接続）を保持する、私たちのアプリケーションの本体です
type Api struct {
	store *Store
}

func main() {
	// .envからDB接続文字列を取得
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("環境変数 DATABASE_URL を設定してください")
	}

	// データベース接続プールを作成
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		log.Fatalf("データベースURLの解析に失敗しました: %v\n", err)
	}
	// "prepared statement"エラーを回避するための、より確実な設定
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	dbpool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("データベースへの接続に失敗しました: %v\n", err)
	}

	defer dbpool.Close() // アプリケーション終了時に接続を閉じる

	log.Println("Successfully initialized Supabase client!") // 接続準備ができたことをログに出力

	store := NewStore(dbpool)
	api := &Api{store: store}

	// ルーティング設定を、apiのメソッドを呼び出すように変更
	mux := http.NewServeMux()
	mux.HandleFunc("/", api.healthCheckHandler)
	mux.HandleFunc("/api/beans", api.getBeansHandler)
	mux.HandleFunc("/api/beans/{id}", api.getBeanHandler)

	// CORS設定
	handler := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	}).Handler(mux)

	fmt.Println("Backend server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
