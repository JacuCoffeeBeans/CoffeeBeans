package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// mainアプリケーション本体を保持するグローバル変数
var testApi *Api

// TestMainは、パッケージ内の全てのテストが実行される前に一度だけ呼ばれる特別な関数
func TestMain(m *testing.M) {
	log.Println("テスト用のデータベース接続をセットアップします...")

	// .envファイルから環境変数を読み込む
	if err := godotenv.Load("../.env"); err != nil {
		// log.Fatalからlog.Printfに戻す。ファイルがなくても環境変数があればOK。
		log.Printf("Warning: .env file not found, relying on environment variables")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("テストを実行するにはDATABASE_URLを設定してください")
	}

	// データベース接続プールを一度だけ作成
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

	// 全てのテストで共有するApiインスタンスを作成
	store := NewStore(dbpool)
	testApi = &Api{store: store}

	// ここで全てのテストが実行される
	exitCode := m.Run()

	// 全てのテストが終わった後に、接続プールを閉じる
	log.Println("テスト用のデータベース接続をクローズします...")
	dbpool.Close()

	// テストを終了
	os.Exit(exitCode)
}

// TestGetBeansHandlerは、DBから豆リストを取得するAPIの統合テストです
func TestGetBeansHandler(t *testing.T) {
	// TestMainで作成した共有インスタンスを使用
	api := testApi

	req, _ := http.NewRequest("GET", "/api/beans", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(api.getBeansHandler)
	handler.ServeHTTP(rr, req)

	// ステータスコードの検証
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusOK)
	}

	// レスポンスボディの検証
	var beans []Bean
	if err := json.NewDecoder(rr.Body).Decode(&beans); err != nil {
		t.Fatalf("レスポンスボディのJSONデコードに失敗しました: %v", err)
	}
}

// TestGetBeanHandlerは、DBから特定の豆を取得するAPIの統合テストです
func TestGetBeanHandler(t *testing.T) {
	// TestMainで作成した共有インスタンスを使用
	api := testApi

	t.Run("正常系: 存在するID", func(t *testing.T) {
		// 事前にSupabaseにID=1のデータが存在することを前提とします
		req, _ := http.NewRequest("GET", "/api/beans/1", nil)
		req.SetPathValue("id", "1")
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(api.getBeanHandler)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("異常系: 存在しないID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/beans/9999", nil)
		req.SetPathValue("id", "9999")
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(api.getBeanHandler)
		handler.ServeHTTP(rr, req)

		// pgxでは、Scan対象の行がない場合エラーが返るので、500エラーになるのが期待値
		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusInternalServerError)
		}
	})
}
