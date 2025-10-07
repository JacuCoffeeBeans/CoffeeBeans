package main

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// データベース接続プールを保持するグローバル変数
var testDbpool *pgxpool.Pool

// TestMainは、パッケージ内の全てのテストが実行される前に一度だけ呼ばれる特別な関数
func TestMain(m *testing.M) {
	log.Println("テスト用のデータベース接続をセットアップします...")

	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("Warning: .env file not found, relying on environment variables")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("テストを実行するにはDATABASE_URLを設定してください")
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		log.Fatalf("データベースURLの解析に失敗しました: %v", err)
	}
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	testDbpool, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("データベースへの接続に失敗しました: %v", err)
	}

	dummyUserID := "00000000-0000-0000-0000-000000000000"
	otherDummyUserID := "11111111-1111-1111-1111-111111111111"
	_, err = testDbpool.Exec(context.Background(), `
			INSERT INTO auth.users (id, email, encrypted_password, created_at, updated_at)
			VALUES ($1, $2, $3, NOW(), NOW()), ($4, $5, $6, NOW(), NOW())
			ON CONFLICT (id) DO NOTHING;
		`, dummyUserID, "test@example.com", "dummy_password", otherDummyUserID, "other@example.com", "dummy_password")
	if err != nil {
		log.Fatalf("テスト用ダミーユーザーの挿入に失敗しました: %v", err)
	}

	exitCode := m.Run()

	_, err = testDbpool.Exec(context.Background(), `DELETE FROM auth.users WHERE id = ANY($1)`, []string{dummyUserID, otherDummyUserID})
	if err != nil {
		log.Printf("Warning: テスト用ダミーユーザーの削除に失敗しました: %v", err)
	}
	log.Println("テスト用のデータベース接続をクローズします...")
	testDbpool.Close()

	os.Exit(exitCode)
}
