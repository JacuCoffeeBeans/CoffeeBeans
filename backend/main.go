// backend/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

// Api はStore（DB接続）を保持する、私たちのアプリケーションの本体です
type Api struct {
	store *Store
}

func main() {

	// アプリケーション起動時に.envファイルを読み込む
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, relying on environment variables")
	}

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
	// "prepared statement"エラーを回避するための設定
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	dbpool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("データベースへの接続に失敗しました: %v\n", err)
	}

	defer dbpool.Close() // アプリケーション終了時に接続を閉じる

	log.Println("Successfully initialized Supabase client!") // 接続準備ができたことをログに出力

	store := NewStore(dbpool)
	api := &Api{store: store}

	// ルーティング設定
	// 1. 各URLで何をするかのハンドラを定義する

	// "/" へのリクエスト担当
	healthCheckHandler := http.HandlerFunc(api.healthCheckHandler)

	// "/api/beans" へのリクエスト担当 (GETとPOSTを振り分ける)
	beansHandler := http.HandlerFunc(api.beansHandler)

	// "/api/beans/{id}" へのリクエスト担当 (GET, PUT, DELETEなどを振り分ける)
	beanDetailHandler := http.HandlerFunc(api.beanDetailHandler)

	// "/api/my/beans" へのリクエスト担当
	myBeansHandler := http.HandlerFunc(api.getMyBeansHandler)

	// "/api/cart/items" へのリクエスト担当
	addCartItemHandler := http.HandlerFunc(api.addCartItemHandler)

	// 2. URLとハンドラを結びつける
	mux := http.NewServeMux()
	mux.Handle("/", healthCheckHandler)

	// `/api/beans` と `/api/beans/{id}` の両方をミドルウェアで保護する
	mux.Handle("/api/beans", jwtAuthMiddleware(beansHandler))
	mux.Handle("/api/beans/{id}", jwtAuthMiddleware(beanDetailHandler))
	mux.Handle("/api/my/beans", jwtAuthMiddleware(myBeansHandler))
	mux.Handle("/api/cart/items", jwtAuthMiddleware(addCartItemHandler))

	// CORS設定
	handler := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	}).Handler(mux)

	fmt.Println("Backend server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
