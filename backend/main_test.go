package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
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

	// .envファイルから環境変数を読み込む
	if err := godotenv.Load("../.env"); err != nil {
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
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	testDbpool, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("データベースへの接続に失敗しました: %v\n", err)
	}

	// ここで全てのテストが実行される
	exitCode := m.Run()

	// 全てのテストが終わった後に、接続プールを閉じる
	log.Println("テスト用のデータベース接続をクローズします...")
	testDbpool.Close()

	// テストを終了
	os.Exit(exitCode)
}

// TestGetBeansHandlerは、DBから豆リストを取得するAPIの統合テストです
func TestGetBeansHandler(t *testing.T) {
	ctx := context.Background()
	tx, err := testDbpool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	store := NewStore(tx)
	api := &Api{store: store}

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
	ctx := context.Background()
	tx, err := testDbpool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	// テストデータを作成
	// このテストはトランザクション内で実行されるため、ここで作成したデータはテスト終了後に自動的にロールバックされます。
	bean := &Bean{Name: "Test Bean for Get", Origin: "Test Origin", Price: 1000, Process: "washed", RoastProfile: "medium"}
	createdBean, err := NewStore(tx).CreateBean(ctx, bean)
	if err != nil {
		t.Fatalf("テストデータの作成に失敗しました: %v", err)
	}

	store := NewStore(tx)
	api := &Api{store: store}

	t.Run("正常系: 存在するID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/beans/"+strconv.Itoa(createdBean.ID), nil)
		req.SetPathValue("id", strconv.Itoa(createdBean.ID))
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

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusInternalServerError)
		}
	})
}

// TestCreateBeanHandler は、新しい豆を作成するAPIの統合テストです
func TestCreateBeanHandler(t *testing.T) {
	ctx := context.Background()
	tx, err := testDbpool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	store := NewStore(tx)
	api := &Api{store: store}
	handler := http.HandlerFunc(api.beansHandler) // テスト対象を、認証チェックを含むbeansHandlerに変更

	t.Run("正常系: 新しい豆を作成", func(t *testing.T) {
		// テスト用のリクエストボディを作成
		body := `{"name": "Test Bean", "origin": "Test Origin", "price": 1000, "process": "Washed", "roast_profile": "Medium"}`
		req, _ := http.NewRequest("POST", "/api/beans", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		// 認証情報をコンテキストに追加
		ctxWithUser := context.WithValue(req.Context(), "userID", "test-user-id")
		req = req.WithContext(ctxWithUser)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		// ステータスコードの検証
		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusCreated)
		}

		// レスポンスボディの検証
		var bean Bean
		if err := json.NewDecoder(rr.Body).Decode(&bean); err != nil {
			t.Fatalf("レスポンスボディのJSONデコードに失敗しました: %v", err)
		}
		if bean.Name != "Test Bean" {
			t.Errorf("期待と異なる豆の名前です: got %v want %v", bean.Name, "Test Bean")
		}
	})

	t.Run("異常系: 認証なし", func(t *testing.T) {
		// テスト用のリクエストボディを作成
		body := `{"name": "Unauthorized Bean", "origin": "Test Origin"}`
		req, _ := http.NewRequest("POST", "/api/beans", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		// 認証情報はコンテキストに追加しない

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		// ステータスコードの検証
		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusUnauthorized)
		}
	})
}