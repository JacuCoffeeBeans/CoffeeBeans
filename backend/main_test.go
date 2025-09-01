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
		log.Fatalf("データベースURLの解析に失敗しました: %v", err)
	}
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	testDbpool, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("データベースへの接続に失敗しました: %v", err)
	}

	// テスト用のダミーユーザーを挿入
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

		// ここで全てのテストが実行される
		exitCode := m.Run()

		// 全てのテストが終わった後に、ダミーユーザーを削除し、接続プールを閉じる
		_, err = testDbpool.Exec(context.Background(), `DELETE FROM auth.users WHERE id = ANY($1)`, []string{dummyUserID, otherDummyUserID})
		if err != nil {
			log.Printf("Warning: テスト用ダミーユーザーの削除に失敗しました: %v", err)
		}
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
	bean := &Bean{Name: "Test Bean for Get", Origin: "Test Origin", Price: 1000, Process: "washed", RoastProfile: "medium", UserID: "00000000-0000-0000-0000-000000000000"}
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
		ctxWithUser := context.WithValue(req.Context(), "userID", "00000000-0000-0000-0000-000000000000")
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

// TestUpdateBeanHandler は、既存の豆を更新するAPIの統合テストです
func TestUpdateBeanHandler(t *testing.T) {
	ctx := context.Background()
	tx, err := testDbpool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) // テスト終了時にロールバック

	store := NewStore(tx)
	api := &Api{store: store}
	// beanDetailHandlerは、PUTリクエストの際に認証チェックを行うので、テスト対象はbeanDetailHandler
	handler := http.HandlerFunc(api.beanDetailHandler)

	// --- テストデータの準備 ---
	// 所有者となるユーザーID
	ownerUserID := "00000000-0000-0000-0000-000000000000"
	// 他のユーザーのID
	otherUserID := "11111111-1111-1111-1111-111111111111"

	// 所有者が作成した豆
	myBean := &Bean{Name: "My Bean", Origin: "My Origin", Process: "washed", RoastProfile: "medium", UserID: ownerUserID}
	createdMyBean, err := store.CreateBean(ctx, myBean)
	if err != nil {
		t.Fatalf("テストデータ（自分の豆）の作成に失敗しました: %v", err)
	}

	// 他のユーザーが作成した豆
	otherBean := &Bean{Name: "Other's Bean", Origin: "Other's Origin", Process: "natural", RoastProfile: "light", UserID: otherUserID}
	createdOtherBean, err := store.CreateBean(ctx, otherBean)
	if err != nil {
		t.Fatalf("テストデータ（他人の豆）の作成に失敗しました: %v", err)
	}

	t.Run("正常系: 自分の豆を更新", func(t *testing.T) {
		updateBody := `{"name": "Updated Name", "origin": "Updated Origin", "process": "honey", "roast_profile": "medium"}`
		url := "/api/beans/" + strconv.Itoa(createdMyBean.ID)
		req, _ := http.NewRequest(http.MethodPut, url, strings.NewReader(updateBody))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", strconv.Itoa(createdMyBean.ID))

		// 認証情報（所有者）をコンテキストに追加
		ctxWithOwner := context.WithValue(req.Context(), "userID", ownerUserID)
		req = req.WithContext(ctxWithOwner)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("期待と異なるステータスコードです: got %v want %v, body: %s", status, http.StatusOK, rr.Body.String())
		}

		var updatedBean Bean
		if err := json.NewDecoder(rr.Body).Decode(&updatedBean); err != nil {
			t.Fatalf("レスポンスボディのJSONデコードに失敗しました: %v", err)
		}
		if updatedBean.Name != "Updated Name" {
			t.Errorf("期待と異なる豆の名前です: got %v want %v", updatedBean.Name, "Updated Name")
		}
	})

	t.Run("異常系: 他人の豆を更新しようとする", func(t *testing.T) {
		updateBody := `{"name": "Malicious Update", "origin": "malicious", "process": "washed", "roast_profile": "medium"}`
		url := "/api/beans/" + strconv.Itoa(createdOtherBean.ID)
		req, _ := http.NewRequest(http.MethodPut, url, strings.NewReader(updateBody))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", strconv.Itoa(createdOtherBean.ID))

		// 認証情報（自分）をコンテキストに追加
		ctxWithOwner := context.WithValue(req.Context(), "userID", ownerUserID)
		req = req.WithContext(ctxWithOwner)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		// 他人のリソースを更新しようとした場合は、Not Foundを返すのが一般的
		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusNotFound)
		}
	})

	t.Run("異常系: 認証なしで更新しようとする", func(t *testing.T) {
		updateBody := `{"name": "Unauthorized Update"}`
		url := "/api/beans/" + strconv.Itoa(createdMyBean.ID)
		req, _ := http.NewRequest(http.MethodPut, url, strings.NewReader(updateBody))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", strconv.Itoa(createdMyBean.ID))

		// 認証情報を追加しない

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusUnauthorized)
		}
	})

	t.Run("異常系: 存在しないIDを更新しようとする", func(t *testing.T) {
		updateBody := `{"name": "Non-existent Update", "origin": "non-existent", "process": "washed", "roast_profile": "medium"}`
		url := "/api/beans/99999"
		req, _ := http.NewRequest(http.MethodPut, url, strings.NewReader(updateBody))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", "99999")

		// 認証情報をコンテキストに追加
		ctxWithOwner := context.WithValue(req.Context(), "userID", ownerUserID)
		req = req.WithContext(ctxWithOwner)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusNotFound)
		}
	})
}

// TestGetMyBeansHandler は、認証されたユーザー自身の豆リストを取得するAPIの統合テストです
func TestGetMyBeansHandler(t *testing.T) {
	ctx := context.Background()
	tx, err := testDbpool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) // テスト終了時にロールバック

	store := NewStore(tx)
	api := &Api{store: store}
	handler := http.HandlerFunc(api.getMyBeansHandler)

	// --- テストデータの準備 ---
	ownerUserID := "00000000-0000-0000-0000-000000000000"
	otherUserID := "11111111-1111-1111-1111-111111111111"

	// 所有者が2つの豆を作成
	_, err = store.CreateBean(ctx, &Bean{Name: "My Bean 1", Origin: "Origin 1", Process: "washed", RoastProfile: "medium", UserID: ownerUserID})
	if err != nil {
		t.Fatalf("テストデータの作成に失敗しました: %v", err)
	}
	_, err = store.CreateBean(ctx, &Bean{Name: "My Bean 2", Origin: "Origin 2", Process: "natural", RoastProfile: "light", UserID: ownerUserID})
	if err != nil {
		t.Fatalf("テストデータの作成に失敗しました: %v", err)
	}

	// 他のユーザーが1つの豆を作成
	_, err = store.CreateBean(ctx, &Bean{Name: "Other's Bean", Origin: "Other Origin", Process: "honey", RoastProfile: "medium", UserID: otherUserID})
	if err != nil {
		t.Fatalf("テストデータの作成に失敗しました: %v", err)
	}

	t.Run("正常系: 自分の豆リストを取得", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/my/beans", nil)

		// 認証情報（所有者）をコンテキストに追加
		ctxWithOwner := context.WithValue(req.Context(), "userID", ownerUserID)
		req = req.WithContext(ctxWithOwner)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("期待と異なるステータスコードです: got %v want %v, body: %s", status, http.StatusOK, rr.Body.String())
		}

		var beans []Bean
		if err := json.NewDecoder(rr.Body).Decode(&beans); err != nil {
			t.Fatalf("レスポンスボディのJSONデコードに失敗しました: %v", err)
		}

		// 取得した豆の数が2であることを確認
		if len(beans) != 2 {
			t.Errorf("期待と異なる数の豆が返されました: got %d want %d", len(beans), 2)
		}

		// 返された豆の所有者がすべて自分であることを確認
		for _, b := range beans {
			if b.UserID != ownerUserID {
				t.Errorf("他のユーザーの豆が含まれています: userID %s", b.UserID)
			}
		}
	})

	t.Run("異常系: 認証なしで取得しようとする", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/my/beans", nil)

		// 認証情報を追加しない

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusUnauthorized)
		}
	})
}

// TestDeleteBeanHandler は、既存の豆を削除するAPIの統合テストです
func TestDeleteBeanHandler(t *testing.T) {
	ctx := context.Background()
	tx, err := testDbpool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) // テスト終了時にロールバック

	store := NewStore(tx)
	api := &Api{store: store}
	// beanDetailHandlerは、DELETEリクエストの際に認証チェックを行うので、テスト対象はbeanDetailHandler
	handler := http.HandlerFunc(api.beanDetailHandler)

	// --- テストデータの準備 ---
	// 所有者となるユーザーID
	ownerUserID := "00000000-0000-0000-0000-000000000000"
	// 他のユーザーのID
	otherUserID := "11111111-1111-1111-1111-111111111111"

	// 所有者が作成した豆（削除対象）
	myBean := &Bean{Name: "My Deletable Bean", Origin: "My Origin", Process: "washed", RoastProfile: "medium", UserID: ownerUserID}
	createdMyBean, err := store.CreateBean(ctx, myBean)
	if err != nil {
		t.Fatalf("テストデータ（自分の豆）の作成に失敗しました: %v", err)
	}

	// 他のユーザーが作成した豆（削除されない対象）
	otherBean := &Bean{Name: "Other's Bean", Origin: "Other's Origin", Process: "natural", RoastProfile: "light", UserID: otherUserID}
	createdOtherBean, err := store.CreateBean(ctx, otherBean)
	if err != nil {
		t.Fatalf("テストデータ（他人の豆）の作成に失敗しました: %v", err)
	}

	t.Run("正常系: 自分の豆を削除", func(t *testing.T) {
		url := "/api/beans/" + strconv.Itoa(createdMyBean.ID)
		req, _ := http.NewRequest(http.MethodDelete, url, nil)
		req.SetPathValue("id", strconv.Itoa(createdMyBean.ID))

		// 認証情報（所有者）をコンテキストに追加
		ctxWithOwner := context.WithValue(req.Context(), "userID", ownerUserID)
		req = req.WithContext(ctxWithOwner)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNoContent {
			t.Errorf("期待と異なるステータスコードです: got %v want %v, body: %s", status, http.StatusNoContent, rr.Body.String())
		}

		// DBから削除されていることを確認
		_, err := store.GetBeanByID(ctx, createdMyBean.ID)
		if err == nil {
			t.Errorf("豆がデータベースから削除されていません")
		}
	})

	t.Run("異常系: 他人の豆を削除しようとする", func(t *testing.T) {
		url := "/api/beans/" + strconv.Itoa(createdOtherBean.ID)
		req, _ := http.NewRequest(http.MethodDelete, url, nil)
		req.SetPathValue("id", strconv.Itoa(createdOtherBean.ID))

		// 認証情報（自分）をコンテキストに追加
		ctxWithOwner := context.WithValue(req.Context(), "userID", ownerUserID)
		req = req.WithContext(ctxWithOwner)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		// 他人のリソースを削除しようとした場合は、Not Foundを返す
		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusNotFound)
		}
	})

	t.Run("異常系: 認証なしで削除しようとする", func(t *testing.T) {
		url := "/api/beans/" + strconv.Itoa(createdMyBean.ID)
		req, _ := http.NewRequest(http.MethodDelete, url, nil)
		req.SetPathValue("id", strconv.Itoa(createdMyBean.ID))

		// 認証情報を追加しない

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusUnauthorized)
		}
	})

	t.Run("異常系: 存在しないIDを削除しようとする", func(t *testing.T) {
		url := "/api/beans/99999"
		req, _ := http.NewRequest(http.MethodDelete, url, nil)
		req.SetPathValue("id", "99999")

		// 認証情報をコンテキストに追加
		ctxWithOwner := context.WithValue(req.Context(), "userID", ownerUserID)
		req = req.WithContext(ctxWithOwner)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusNotFound)
		}
	})
}