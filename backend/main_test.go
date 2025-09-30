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
)

// データベース接続プールを保持するグローバル変数
var testDbpool *pgxpool.Pool

// TestMainは、パッケージ内の全てのテストが実行される前に一度だけ呼ばれる特別な関数
func TestMain(m *testing.M) {
	log.Println("テスト用のデータベース接続をセットアップします...")

	// .envファイルから環境変数を読み込む
	// if err := godotenv.Load("../.env"); err != nil {
	// 	log.Printf("Warning: .env file not found, relying on environment variables")
	// }
	// docker-compose.ymlのenv_fileで環境変数が設定されるため、↑は不要

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
		ctxWithUser := context.WithValue(req.Context(), userIDKey, "00000000-0000-0000-0000-000000000000")
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
		ctxWithOwner := context.WithValue(req.Context(), userIDKey, ownerUserID)
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
		ctxWithOwner := context.WithValue(req.Context(), userIDKey, ownerUserID)
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
		ctxWithOwner := context.WithValue(req.Context(), userIDKey, ownerUserID)
		req = req.WithContext(ctxWithOwner)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusNotFound)
		}
	})
}

// TestAddCartItemHandler は、カートに商品を追加するAPIの統合テストです
func TestAddCartItemHandler(t *testing.T) {
	ctx := context.Background()
	tx, err := testDbpool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	store := NewStore(tx)
	api := &Api{store: store}
	handler := http.HandlerFunc(api.addCartItemHandler)

	// --- テストデータの準備 ---
	userID := "00000000-0000-0000-0000-000000000000"
	// テスト用の豆を作成
	bean := &Bean{Name: "Test Bean for Cart", Origin: "Cart Origin", Process: "washed", RoastProfile: "medium", UserID: userID}
	createdBean, err := store.CreateBean(ctx, bean)
	if err != nil {
		t.Fatalf("テストデータの作成に失敗しました: %v", err)
	}

	t.Run("正常系: 新しい商品をカートに追加", func(t *testing.T) {
		body := `{"bean_id": ` + strconv.Itoa(createdBean.ID) + `, "quantity": 2}`
		req, _ := http.NewRequest(http.MethodPost, "/api/cart/items", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		// 認証情報をコンテキストに追加
		ctxWithUser := context.WithValue(req.Context(), userIDKey, userID)
		req = req.WithContext(ctxWithUser)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("期待と異なるステータスコードです: got %v want %v, body: %s", status, http.StatusOK, rr.Body.String())
		}

		var cartItem CartItem
		if err := json.NewDecoder(rr.Body).Decode(&cartItem); err != nil {
			t.Fatalf("レスポンスボディのJSONデコードに失敗しました: %v", err)
		}
		if cartItem.BeanID != createdBean.ID || cartItem.Quantity != 2 {
			t.Errorf("期待と異なるカートアイテムが作成されました: got %+v", cartItem)
		}
	})

	t.Run("正常系: 既存の商品の数量を更新", func(t *testing.T) {
		// このテストケースの独立性を保つために、テスト前にカートの状態をクリーンにする
		_, err := tx.Exec(ctx, "DELETE FROM cart_items WHERE bean_id = $1", createdBean.ID)
		if err != nil {
			t.Fatalf("テストのためにcart_itemsをクリーンアップするのに失敗しました: %v", err)
		}

		// 最初に商品を1つ追加しておく
		firstBody := `{"bean_id": ` + strconv.Itoa(createdBean.ID) + `, "quantity": 1}`
		firstReq, _ := http.NewRequest(http.MethodPost, "/api/cart/items", strings.NewReader(firstBody))
		ctxWithUser := context.WithValue(firstReq.Context(), userIDKey, userID)
		firstReq = firstReq.WithContext(ctxWithUser)
		handler.ServeHTTP(httptest.NewRecorder(), firstReq)

		// 同じ商品をさらに3つ追加
		secondBody := `{"bean_id": ` + strconv.Itoa(createdBean.ID) + `, "quantity": 3}`
		secondReq, _ := http.NewRequest(http.MethodPost, "/api/cart/items", strings.NewReader(secondBody))
		secondReq = secondReq.WithContext(ctxWithUser)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, secondReq)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("期待と異なるステータスコードです: got %v want %v, body: %s", status, http.StatusOK, rr.Body.String())
		}

		var cartItem CartItem
		if err := json.NewDecoder(rr.Body).Decode(&cartItem); err != nil {
			t.Fatalf("レスポンスボディのJSONデコードに失敗しました: %v", err)
		}
		// 1 + 3 = 4 になっているはず
		if cartItem.Quantity != 4 {
			t.Errorf("カートアイテムの数量が正しく更新されていません: got %d want %d", cartItem.Quantity, 4)
		}
	})

	t.Run("異常系: 認証なし", func(t *testing.T) {
		body := `{"bean_id": 1, "quantity": 1}`
		req, _ := http.NewRequest(http.MethodPost, "/api/cart/items", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusUnauthorized)
		}
	})

	t.Run("異常系: 不正なリクエストボディ", func(t *testing.T) {
		body := `{"bean_id": 0, "quantity": 0}` // 不正な値
		req, _ := http.NewRequest(http.MethodPost, "/api/cart/items", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		ctxWithUser := context.WithValue(req.Context(), userIDKey, userID)
		req = req.WithContext(ctxWithUser)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusBadRequest)
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
		ctxWithOwner := context.WithValue(req.Context(), userIDKey, ownerUserID)
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
		ctxWithOwner := context.WithValue(req.Context(), userIDKey, ownerUserID)
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
		ctxWithOwner := context.WithValue(req.Context(), userIDKey, ownerUserID)
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
		ctxWithOwner := context.WithValue(req.Context(), userIDKey, ownerUserID)
		req = req.WithContext(ctxWithOwner)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusNotFound)
		}
	})
}

// TestGetCartHandler は、カートの中身を取得するAPIの統合テストです
func TestGetCartHandler(t *testing.T) {
	ctx := context.Background()
	tx, err := testDbpool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	store := NewStore(tx)
	api := &Api{store: store}
	// getCartHandlerは認証が必要なので、ミドルウェアを通過した後のハンドラを直接テスト
	handler := http.HandlerFunc(api.getCartHandler)

	// --- テストデータの準備 ---
	userID := "00000000-0000-0000-0000-000000000000"
	otherUserID := "11111111-1111-1111-1111-111111111111" // カートが空のユーザー

	// テスト用の豆を作成
	bean, err := store.CreateBean(ctx, &Bean{Name: "Cart Test Bean", Origin: "Test", Price: 1200, Process: "natural", RoastProfile: "light", UserID: userID})
	if err != nil {
		t.Fatalf("テスト用の豆の作成に失敗しました: %v", err)
	}

	// userIDのユーザーのカートに商品を追加
	_, err = store.AddOrUpdateCartItem(ctx, userID, AddCartItemRequest{BeanID: bean.ID, Quantity: 3})
	if err != nil {
		t.Fatalf("テスト用のカートアイテムの作成に失敗しました: %v", err)
	}

	t.Run("正常系: 認証済みでカートに商品あり", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/cart", nil)

		// 認証情報（カートに商品があるユーザー）をコンテキストに追加
		ctxWithUser := context.WithValue(req.Context(), userIDKey, userID)
		req = req.WithContext(ctxWithUser)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("期待と異なるステータスコードです: got %v want %v, body: %s", status, http.StatusOK, rr.Body.String())
		}

		var items []CartItemDetail
		if err := json.NewDecoder(rr.Body).Decode(&items); err != nil {
			t.Fatalf("レスポンスボディのJSONデコードに失敗しました: %v", err)
		}

		if len(items) != 1 {
			t.Fatalf("期待と異なる数の商品が返されました: got %d want %d", len(items), 1)
		}
		if items[0].BeanID != bean.ID || items[0].Quantity != 3 || items[0].Name != "Cart Test Bean" {
			t.Errorf("カートの商品情報が期待と異なります: got %+v", items[0])
		}
	})

	t.Run("正常系: 認証済みでカートが空", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/cart", nil)

		// 認証情報（カートが空のユーザー）をコンテキストに追加
		ctxWithUser := context.WithValue(req.Context(), userIDKey, otherUserID)
		req = req.WithContext(ctxWithUser)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusOK)
		}

		// レスポンスボディが空のJSON配列 "[]" であることを確認
		if body := strings.TrimSpace(rr.Body.String()); body != "[]" {
			t.Errorf("レスポンスボディが空の配列ではありません: got %s", body)
		}
	})

	t.Run("異常系: 認証なし", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/cart", nil)

		// 認証情報をコンテキストに追加しない
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusUnauthorized)
		}
	})
}

// TestUpdateCartItemAPI は、カート内の商品の数量を更新するAPIの統合テストです
func TestUpdateCartItemAPI(t *testing.T) {
	ctx := context.Background()
	tx, err := testDbpool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	store := NewStore(tx)
	api := &Api{store: store}
	handler := http.HandlerFunc(api.cartItemDetailHandler)

	// --- テストデータの準備 ---
	ownerUserID := "00000000-0000-0000-0000-000000000000"
	otherUserID := "11111111-1111-1111-1111-111111111111"

	// テスト用の豆を作成 (自分用)
	myBean, err := store.CreateBean(ctx, &Bean{Name: "Update Cart Test Bean", Origin: "Test", Process: "washed", RoastProfile: "medium", UserID: ownerUserID})
	if err != nil {
		t.Fatalf("テスト用の豆の作成に失敗しました: %v", err)
	}
	// テスト用の豆を作成 (他人用)
	otherBean, err := store.CreateBean(ctx, &Bean{Name: "Other's Cart Test Bean", Origin: "Test", Process: "natural", RoastProfile: "light", UserID: otherUserID})
	if err != nil {
		t.Fatalf("テスト用の豆（他人用）の作成に失敗しました: %v", err)
	}

	// 所有者のカートに商品を追加
	myCartItem, err := store.AddOrUpdateCartItem(ctx, ownerUserID, AddCartItemRequest{BeanID: myBean.ID, Quantity: 1})
	if err != nil {
		t.Fatalf("テスト用のカートアイテム（自分）の作成に失敗しました: %v", err)
	}

	// 他のユーザーのカートに商品を追加
	otherCartItem, err := store.AddOrUpdateCartItem(ctx, otherUserID, AddCartItemRequest{BeanID: otherBean.ID, Quantity: 2})
	if err != nil {
		t.Fatalf("テスト用のカートアイテム（他人）の作成に失敗しました: %v", err)
	}

	t.Run("正常系: 自分のカートアイテムの数量を更新", func(t *testing.T) {
		updateBody := `{"quantity": 5}`
		url := "/api/cart/items/" + myCartItem.ID
		req, _ := http.NewRequest(http.MethodPut, url, strings.NewReader(updateBody))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", myCartItem.ID)

		// 認証情報（所有者）をコンテキストに追加
		ctxWithOwner := context.WithValue(req.Context(), userIDKey, ownerUserID)
		req = req.WithContext(ctxWithOwner)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("期待と異なるステータスコードです: got %v want %v, body: %s", status, http.StatusOK, rr.Body.String())
		}

		var updatedItem CartItem
		if err := json.NewDecoder(rr.Body).Decode(&updatedItem); err != nil {
			t.Fatalf("レスポンスボディのJSONデコードに失敗しました: %v", err)
		}
		if updatedItem.Quantity != 5 {
			t.Errorf("数量が正しく更新されていません: got %d want %d", updatedItem.Quantity, 5)
		}
	})

	t.Run("異常系: 他人のカートアイテムを更新しようとする", func(t *testing.T) {
		updateBody := `{"quantity": 10}`
		url := "/api/cart/items/" + otherCartItem.ID
		req, _ := http.NewRequest(http.MethodPut, url, strings.NewReader(updateBody))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", otherCartItem.ID)

		// 認証情報（自分）をコンテキストに追加
		ctxWithOwner := context.WithValue(req.Context(), userIDKey, ownerUserID)
		req = req.WithContext(ctxWithOwner)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusNotFound)
		}
	})

	t.Run("異常系: 認証なしで更新しようとする", func(t *testing.T) {
		updateBody := `{"quantity": 3}`
		url := "/api/cart/items/" + myCartItem.ID
		req, _ := http.NewRequest(http.MethodPut, url, strings.NewReader(updateBody))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", myCartItem.ID)

		// 認証情報を追加しない
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusUnauthorized)
		}
	})

	t.Run("異常系: 数量に0を指定する", func(t *testing.T) {
		updateBody := `{"quantity": 0}`
		url := "/api/cart/items/" + myCartItem.ID
		req, _ := http.NewRequest(http.MethodPut, url, strings.NewReader(updateBody))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", myCartItem.ID)

		ctxWithOwner := context.WithValue(req.Context(), userIDKey, ownerUserID)
		req = req.WithContext(ctxWithOwner)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusBadRequest)
		}
	})
}

// TestDeleteCartItemAPI は、カートから商品を削除するAPIの統合テストです
func TestDeleteCartItemAPI(t *testing.T) {
	ctx := context.Background()
	tx, err := testDbpool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	store := NewStore(tx)
	api := &Api{store: store}
	handler := http.HandlerFunc(api.cartItemDetailHandler)

	// --- テストデータの準備 ---
	ownerUserID := "00000000-0000-0000-0000-000000000000"
	otherUserID := "11111111-1111-1111-1111-111111111111"

	// テスト用の豆を作成 (自分用)
	myBean, err := store.CreateBean(ctx, &Bean{Name: "Delete Cart Test Bean", Origin: "Test", Process: "washed", RoastProfile: "medium", UserID: ownerUserID})
	if err != nil {
		t.Fatalf("テスト用の豆の作成に失敗しました: %v", err)
	}
	// テスト用の豆を作成 (他人用)
	otherBean, err := store.CreateBean(ctx, &Bean{Name: "Other's Delete Cart Test Bean", Origin: "Test", Process: "natural", RoastProfile: "light", UserID: otherUserID})
	if err != nil {
		t.Fatalf("テスト用の豆（他人用）の作成に失敗しました: %v", err)
	}

	// 所有者のカートに商品を追加（削除対象）
	myCartItem, err := store.AddOrUpdateCartItem(ctx, ownerUserID, AddCartItemRequest{BeanID: myBean.ID, Quantity: 1})
	if err != nil {
		t.Fatalf("テスト用のカートアイテム（自分）の作成に失敗しました: %v", err)
	}

	// 他のユーザーのカートに商品を追加（削除されない対象）
	otherCartItem, err := store.AddOrUpdateCartItem(ctx, otherUserID, AddCartItemRequest{BeanID: otherBean.ID, Quantity: 2})
	if err != nil {
		t.Fatalf("テスト用のカートアイテム（他人）の作成に失敗しました: %v", err)
	}

	t.Run("正常系: 自分のカートアイテムを削除", func(t *testing.T) {
		url := "/api/cart/items/" + myCartItem.ID
		req, _ := http.NewRequest(http.MethodDelete, url, nil)
		req.SetPathValue("id", myCartItem.ID)

		// 認証情報（所有者）をコンテキストに追加
		ctxWithOwner := context.WithValue(req.Context(), userIDKey, ownerUserID)
		req = req.WithContext(ctxWithOwner)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNoContent {
			t.Errorf("期待と異なるステータスコードです: got %v want %v, body: %s", status, http.StatusNoContent, rr.Body.String())
		}

		// DBから削除されていることを確認
		// ユーザーのカートを取得し、アイテムが存在しないことを確認する
		items, err := store.GetCartItemsByUserID(ctx, ownerUserID)
		if err != nil {
			t.Fatalf("カートアイテムの取得に失敗しました: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("カートアイテムがデータベースから削除されていません")
		}
	})

	t.Run("異常系: 他人のカートアイテムを削除しようとする", func(t *testing.T) {
		url := "/api/cart/items/" + otherCartItem.ID
		req, _ := http.NewRequest(http.MethodDelete, url, nil)
		req.SetPathValue("id", otherCartItem.ID)

		// 認証情報（自分）をコンテキストに追加
		ctxWithOwner := context.WithValue(req.Context(), userIDKey, ownerUserID)
		req = req.WithContext(ctxWithOwner)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusNotFound)
		}
	})

	t.Run("異常系: 認証なしで削除しようとする", func(t *testing.T) {
		// このテストケースのために、削除されていない新しいアイテムを準備
		tempItem, _ := store.AddOrUpdateCartItem(ctx, ownerUserID, AddCartItemRequest{BeanID: myBean.ID, Quantity: 99})
		url := "/api/cart/items/" + tempItem.ID
		req, _ := http.NewRequest(http.MethodDelete, url, nil)
		req.SetPathValue("id", tempItem.ID)

		// 認証情報を追加しない
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusUnauthorized)
		}
	})

	t.Run("異常系: 存在しないIDを削除しようとする", func(t *testing.T) {
		// 存在しない数値形式のID
		nonExistentID := "999999"
		url := "/api/cart/items/" + nonExistentID
		req, _ := http.NewRequest(http.MethodDelete, url, nil)
		req.SetPathValue("id", nonExistentID)

		// 認証情報をコンテキストに追加
		ctxWithOwner := context.WithValue(req.Context(), userIDKey, ownerUserID)
		req = req.WithContext(ctxWithOwner)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusNotFound)
		}
	})
}

// TestCreatePaymentIntentHandler は、StripeのPaymentIntentを作成するAPIの統合テストです
func TestCreatePaymentIntentHandler(t *testing.T) {
	ctx := context.Background()
	tx, err := testDbpool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	store := NewStore(tx)
	api := &Api{store: store}
	handler := http.HandlerFunc(api.createPaymentIntentHandler)

	// --- テストデータの準備 ---
	userID := "00000000-0000-0000-0000-000000000000"
	emptyCartUserID := "11111111-1111-1111-1111-111111111111"

	// テスト用の豆を作成
	bean1, err := store.CreateBean(ctx, &Bean{Name: "PI Test Bean 1", Origin: "Test", Price: 1000, Process: "washed", RoastProfile: "medium", UserID: userID})
	if err != nil {
		t.Fatalf("テスト用の豆1の作成に失敗しました: %v", err)
	}
	bean2, err := store.CreateBean(ctx, &Bean{Name: "PI Test Bean 2", Origin: "Test", Price: 500, Process: "natural", RoastProfile: "light", UserID: userID})
	if err != nil {
		t.Fatalf("テスト用の豆2の作成に失敗しました: %v", err)
	}

	// userIDのユーザーのカートに商品を追加
	_, err = store.AddOrUpdateCartItem(ctx, userID, AddCartItemRequest{BeanID: bean1.ID, Quantity: 2}) // 1000 * 2 = 2000
	if err != nil {
		t.Fatalf("テスト用のカートアイテム1の作成に失敗しました: %v", err)
	}
	_, err = store.AddOrUpdateCartItem(ctx, userID, AddCartItemRequest{BeanID: bean2.ID, Quantity: 3}) // 500 * 3 = 1500
	if err != nil {
		t.Fatalf("テスト用のカートアイテム2の作成に失敗しました: %v", err)
	}
	// 合計金額は 2000 + 1500 = 3500円

	t.Run("正常系: カートに商品がある場合にclient_secretを取得", func(t *testing.T) {
		// .envからStripeキーを読み込めていないとテストが失敗するので注意
		if os.Getenv("STRIPE_SECRET_KEY") == "" {
			t.Skip("STRIPE_SECRET_KEYが設定されていないため、テストをスキップします")
		}

		req, _ := http.NewRequest(http.MethodPost, "/api/checkout/payment-intent", nil)

		// 認証情報をコンテキストに追加
		ctxWithUser := context.WithValue(req.Context(), userIDKey, userID)
		req = req.WithContext(ctxWithUser)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("期待と異なるステータスコードです: got %v want %v, body: %s", status, http.StatusOK, rr.Body.String())
		}

		var response map[string]string
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("レスポンスボディのJSONデコードに失敗しました: %v", err)
		}

		clientSecret, ok := response["client_secret"]
		if !ok {
			t.Errorf("レスポンスにclient_secretが含まれていません")
		}
		if !strings.HasPrefix(clientSecret, "pi_") {
			t.Errorf("client_secretの形式が正しくありません: got %s", clientSecret)
		}
	})

	t.Run("異常系: カートが空の場合", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/checkout/payment-intent", nil)

		// カートが空のユーザーで認証
		ctxWithUser := context.WithValue(req.Context(), userIDKey, emptyCartUserID)
		req = req.WithContext(ctxWithUser)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("期待と異なるステータスコ��ドです: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("異常系: 認証なし", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/checkout/payment-intent", nil)

		// 認証情報なし
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusUnauthorized)
		}
	})

	t.Run("異常系: POST以外のメソッド", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/checkout/payment-intent", nil)

		ctxWithUser := context.WithValue(req.Context(), userIDKey, userID)
		req = req.WithContext(ctxWithUser)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusMethodNotAllowed {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusMethodNotAllowed)
		}
	})
}
