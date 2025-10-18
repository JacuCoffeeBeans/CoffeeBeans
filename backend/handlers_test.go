package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
	"net"
	"crypto/tls"

	"github.com/stretchr/testify/assert"
)

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

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusOK)
	}

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
	handler := http.HandlerFunc(api.beansHandler)

	t.Run("正常系: 新しい豆を作成", func(t *testing.T) {
		body := `{"name": "Test Bean", "origin": "Test Origin", "price": 1000, "process": "Washed", "roast_profile": "Medium"}`
		req, _ := http.NewRequest("POST", "/api/beans", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		ctxWithUser := context.WithValue(req.Context(), userIDKey, "00000000-0000-0000-0000-000000000000")
		req = req.WithContext(ctxWithUser)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusCreated)
		}

		var bean Bean
		if err := json.NewDecoder(rr.Body).Decode(&bean); err != nil {
			t.Fatalf("レスポンスボディのJSONデコードに失敗しました: %v", err)
		}
		if bean.Name != "Test Bean" {
			t.Errorf("期待と異なる豆の名前です: got %v want %v", bean.Name, "Test Bean")
		}
	})

	t.Run("異常系: 認証なし", func(t *testing.T) {
		body := `{"name": "Unauthorized Bean", "origin": "Test Origin"}`
		req, _ := http.NewRequest("POST", "/api/beans", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

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
	defer tx.Rollback(ctx)

	store := NewStore(tx)
	api := &Api{store: store}
	handler := http.HandlerFunc(api.beanDetailHandler)

	ownerUserID := "00000000-0000-0000-0000-000000000000"
	otherUserID := "11111111-1111-1111-1111-111111111111"

	myBean := &Bean{Name: "My Bean", Origin: "My Origin", Process: "washed", RoastProfile: "medium", UserID: ownerUserID}
	createdMyBean, err := store.CreateBean(ctx, myBean)
	if err != nil {
		t.Fatalf("テストデータ（自分の豆）の作成に失敗しました: %v", err)
	}

	otherBean := &Bean{Name: "Other's Bean", Origin: "Other's Origin", Process: "natural", RoastProfile: "light", UserID: otherUserID}
	_, err = store.CreateBean(ctx, otherBean)
	if err != nil {
		t.Fatalf("テストデータ（他人の豆）の作成に失敗しました: %v", err)
	}

	t.Run("正常系: 自分の豆を更新", func(t *testing.T) {
		updateBody := `{"name": "Updated Name", "origin": "Updated Origin", "process": "honey", "roast_profile": "medium"}`
		url := "/api/beans/" + strconv.Itoa(createdMyBean.ID)
		req, _ := http.NewRequest(http.MethodPut, url, strings.NewReader(updateBody))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", strconv.Itoa(createdMyBean.ID))

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
}

// testWebhookSecret はテストで使用するWebhookシークレットです
const testWebhookSecret = "whsec_test_secret"

// createTestRequest はv72で利用可能な方法でテストリクエストを生成します
func createTestRequest(t *testing.T, payload string, secret string) *http.Request {
	t.Helper()

	req := httptest.NewRequest("POST", "/api/webhooks/stripe", strings.NewReader(payload))

	timestamp := time.Now()

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d.%s", timestamp.Unix(), payload)))
	signature := hex.EncodeToString(mac.Sum(nil))

	header := fmt.Sprintf("t=%d,v1=%s", timestamp.Unix(), signature)
	req.Header.Set("Stripe-Signature", header)

	return req
}

func TestHandleStripeWebhook(t *testing.T) {
	t.Setenv("STRIPE_WEBHOOK_SECRET", testWebhookSecret)

	// テスト用のApiインスタンスを作成
	store := NewStore(testDbpool)
	api := &Api{store: store, dbpool: testDbpool}

	t.Run("Success with valid signature", func(t *testing.T) {
		payload := `{"id": "evt_test", "type": "payment_intent.succeeded", "data": {"object": {"id": "pi_test", "metadata": {"user_id": "00000000-0000-0000-0000-000000000000"}}}}`
		req := createTestRequest(t, payload, testWebhookSecret)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(api.handleStripeWebhook)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("Fail with no signature header", func(t *testing.T) {
		payload := `{"id": "evt_test", "type": "payment_intent.succeeded", "data": {"object": {}}}`
		req := httptest.NewRequest("POST", "/api/webhooks/stripe", strings.NewReader(payload))

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(api.handleStripeWebhook)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Fail with invalid signature", func(t *testing.T) {
		payload := `{"id": "evt_test", "type": "payment_intent.succeeded", "data": {"object": {}}}`
		req := createTestRequest(t, payload, "wrong_secret")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(api.handleStripeWebhook)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestHandleStripeWebhook_CreateOrder(t *testing.T) {
	t.Setenv("STRIPE_WEBHOOK_SECRET", testWebhookSecret)

	ctx := context.Background()
	// このテストはDBの状態を変更し、それを検証するため、トランザクションではなく実際のDBプールに対して実行します。
	// ただし、テストの独立性を保つために、テストの最後にクリーンアップ処理を入れるべきです。
	// ここでは簡略化のため、テストDBが毎回クリーンな状態から始まることを前提とします。
	store := NewStore(testDbpool)
	api := &Api{store: store, dbpool: testDbpool}

	// --- Arrange ---
	// 1. テスト用のユーザーと豆を作成
	testUserID := "00000000-0000-0000-0000-000000000000"
	bean, err := store.CreateBean(ctx, &Bean{Name: "Test Bean for Order", Origin: "Test", Price: 1500, Process: "washed", RoastProfile: "medium", UserID: testUserID})
	assert.NoError(t, err)
	// テスト終了時に作成したデータを削除
	defer store.DeleteBean(ctx, bean.ID, testUserID)

	// 2. カートに商品を追加
	_, err = store.AddOrUpdateCartItem(ctx, testUserID, AddCartItemRequest{BeanID: bean.ID, Quantity: 2})
	assert.NoError(t, err)
	// このカートアイテムはWebhook内でClearCartされるので、個別の削除は不要

	// --- Act ---
	// 3. Webhookリクエストを送信
	paymentIntentID := "pi_test_" + strconv.FormatInt(time.Now().Unix(), 10)
	payload := fmt.Sprintf(`{"id": "evt_test", "type": "payment_intent.succeeded", "data": {"object": {"id": "%s", "amount": 3000, "currency": "jpy", "metadata": {"user_id": "%s"}, "payment_method_types": ["card"]}}}`, paymentIntentID, testUserID)
	req := createTestRequest(t, payload, testWebhookSecret)

	rr := httptest.NewRecorder()
	api.handleStripeWebhook(rr, req)

	// --- Assert ---
	// 4. 結果を検証
	assert.Equal(t, http.StatusOK, rr.Code)

	// 4a. カートが空になっていることを確認
	cartItems, err := store.GetCartItemsByUserID(ctx, testUserID)
	assert.NoError(t, err)
	assert.Empty(t, cartItems, "カートが空にされていません")

	// 4b. 注文が作成されていることを確認
	order, err := store.GetOrderByPaymentIntentID(ctx, paymentIntentID)
	assert.NoError(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, testUserID, order.UserID)
	assert.Equal(t, "succeeded", order.Status)
	assert.Equal(t, 3000, order.TotalAmount)
	// テスト終了時に作成した注文を削除
	defer testDbpool.Exec(ctx, "DELETE FROM order_items WHERE order_id = $1", order.ID)
	defer testDbpool.Exec(ctx, "DELETE FROM orders WHERE id = $1", order.ID)
}

func TestProfileAPI(t *testing.T) {
	// このテストは複数のサブテストでDBの状態を変更・検証するため、
	// サブテストごとにトランザクションを管理します。
	dummyUserID := "00000000-0000-0000-0000-000000000000"

	t.Run("POST /api/profile - 正常系", func(t *testing.T) {
		ctx := context.Background()
		tx, err := testDbpool.Begin(ctx)
		assert.NoError(t, err)
		defer tx.Rollback(ctx)

		store := NewStore(tx)
		api := &Api{store: store, dbpool: testDbpool}
		handler := http.HandlerFunc(api.profileHandler)

		profileData := `{"display_name": "Test User", "icon_url": "http://example.com/icon.png", "post_code": "123-4567", "address": "Test Address", "about_me": "This is a test user."}`
		req := httptest.NewRequest("POST", "/api/profile", strings.NewReader(profileData))
		req.Header.Set("Content-Type", "application/json")

		ctxWithUser := context.WithValue(req.Context(), userIDKey, dummyUserID)
		req = req.WithContext(ctxWithUser)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var createdProfile Profile
		err = json.NewDecoder(rr.Body).Decode(&createdProfile)
		assert.NoError(t, err)
		assert.Equal(t, dummyUserID, createdProfile.UserID)
		assert.Equal(t, "Test User", createdProfile.DisplayName)
	})

	t.Run("PUT /api/profile - 正常系", func(t *testing.T) {
		ctx := context.Background()
		tx, err := testDbpool.Begin(ctx)
		assert.NoError(t, err)
		defer tx.Rollback(ctx)

		store := NewStore(tx)
		api := &Api{store: store, dbpool: testDbpool}
		handler := http.HandlerFunc(api.profileHandler)

		// 先にプロフィールを作成しておく
		initialProfile := &Profile{UserID: dummyUserID, DisplayName: "Initial User", IconURL: "initial.png", PostCode: "111-1111", Address: "Initial Address", AboutMe: "Initial."}
		_, err = store.CreateProfile(ctx, initialProfile)
		assert.NoError(t, err)

		// 更新処理
		updateData := `{"display_name": "Updated User", "icon_url": "updated.png", "post_code": "222-2222", "address": "Updated Address", "about_me": "Updated."}`
		putReq := httptest.NewRequest("PUT", "/api/profile", strings.NewReader(updateData))
		putReq.Header.Set("Content-Type", "application/json")
		ctxWithUser := context.WithValue(putReq.Context(), userIDKey, dummyUserID)
		putReq = putReq.WithContext(ctxWithUser)

		putRR := httptest.NewRecorder()
		handler.ServeHTTP(putRR, putReq)

		assert.Equal(t, http.StatusOK, putRR.Code)

		var updatedProfile Profile
		err = json.NewDecoder(putRR.Body).Decode(&updatedProfile)
		assert.NoError(t, err)
		assert.Equal(t, "Updated User", updatedProfile.DisplayName)
	})

	t.Run("POST /api/profile - 異常系(重複)", func(t *testing.T) {
		ctx := context.Background()
		tx, err := testDbpool.Begin(ctx)
		assert.NoError(t, err)
		defer tx.Rollback(ctx)

		store := NewStore(tx)
		api := &Api{store: store, dbpool: testDbpool}
		handler := http.HandlerFunc(api.profileHandler)

		// 先にプロフィールを作成しておく
		initialProfile := &Profile{UserID: dummyUserID, DisplayName: "Initial User", IconURL: "initial.png", PostCode: "111-1111", Address: "Initial Address", AboutMe: "Initial."}
		_, err = store.CreateProfile(ctx, initialProfile)
		assert.NoError(t, err)

		// 同じユーザーで再度作成しようとするとエラーになるはず
		duplicateProfileData := `{"display_name": "Duplicate User", "icon_url": "duplicate.png", "post_code": "333-3333", "address": "Duplicate Address", "about_me": "Duplicate."}`
		req := httptest.NewRequest("POST", "/api/profile", strings.NewReader(duplicateProfileData))
		req.Header.Set("Content-Type", "application/json")

		ctxWithUser := context.WithValue(req.Context(), userIDKey, dummyUserID)
		req = req.WithContext(ctxWithUser)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		// PostgreSQLのunique_violationエラー(23505)をハンドラで500として返す想定
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

// TestStripeConnectHandlers はStripe Connect関連のハンドラをテストします
func TestStripeConnectHandlers(t *testing.T) {
	// Stripe APIのモックサーバーをセットアップ
	mockStripeAPIServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/accounts": // アカウント作成
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": "acct_mock12345", "charges_enabled": false, "details_submitted": false}`))
		case "/v1/account_links": // アカウントリンク作成
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"object": "account_link", "url": "https://connect.stripe.com/mock_onboarding"}`))
		case "/v1/accounts/acct_mock12345": // アカウント取得
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": "acct_mock12345", "charges_enabled": true, "details_submitted": true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockStripeAPIServer.Close()

	// Stripe SDKの通信をモックサーバーに向ける
	originalTransport := http.DefaultTransport
	http.DefaultTransport = &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if addr == "api.stripe.com:443" {
				return net.Dial(network, mockStripeAPIServer.Listener.Addr().String())
			}
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		},
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	defer func() {
		http.DefaultTransport = originalTransport
	}()

	dummyUserID := "00000000-0000-0000-0000-000000000000"

	t.Run("POST /api/stripe/connect/account-link - 正常系", func(t *testing.T) {
		ctx := context.Background()
		tx, err := testDbpool.Begin(ctx)
		assert.NoError(t, err)
		defer tx.Rollback(ctx)

		store := NewStore(tx)
		api := &Api{store: store, dbpool: testDbpool}
		handler := http.HandlerFunc(api.createStripeAccountLinkHandler)

		// 先にプロフィールを作成しておく
		_, err = store.CreateProfile(ctx, &Profile{UserID: dummyUserID, DisplayName: "test"})
		assert.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/stripe/connect/account-link", nil)
		ctxWithUser := context.WithValue(req.Context(), userIDKey, dummyUserID)
		req = req.WithContext(ctxWithUser)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var result map[string]string
		err = json.NewDecoder(rr.Body).Decode(&result)
		assert.NoError(t, err)
		assert.Equal(t, "https://connect.stripe.com/mock_onboarding", result["url"])

		// DBにStripe Account IDが保存されたか確認
		profile, err := store.GetProfileByUserID(ctx, dummyUserID)
		assert.NoError(t, err)
		assert.NotNil(t, profile.StripeAccountID)
		assert.Equal(t, "acct_mock12345", *profile.StripeAccountID)
	})

	t.Run("GET /api/stripe/connect/redirect - 正常系", func(t *testing.T) {
		ctx := context.Background()
		tx, err := testDbpool.Begin(ctx)
		assert.NoError(t, err)
		defer tx.Rollback(ctx)

		store := NewStore(tx)
		api := &Api{store: store, dbpool: testDbpool}
		handler := http.HandlerFunc(api.handleStripeConnectRedirectHandler)

		// 先にプロフィールとStripeアカウントIDを作成しておく
		_, err = store.CreateProfile(ctx, &Profile{UserID: dummyUserID, DisplayName: "test"})
		assert.NoError(t, err)
		err = store.UpdateStripeAccount(ctx, dummyUserID, "acct_mock12345", "restricted")
		assert.NoError(t, err)

		req := httptest.NewRequest("GET", "/api/stripe/connect/redirect", nil)
		ctxWithUser := context.WithValue(req.Context(), userIDKey, dummyUserID)
		req = req.WithContext(ctxWithUser)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		// リダイレクトされることを確認
		assert.Equal(t, http.StatusFound, rr.Code)
		assert.Equal(t, "http://localhost:5173/my/beans?stripe_connect=success", rr.Header().Get("Location"))

		// DBのステータスが更新されたか確認
		profile, err := store.GetProfileByUserID(ctx, dummyUserID)
		assert.NoError(t, err)
		assert.NotNil(t, profile.StripeAccountStatus)
		assert.Equal(t, "enabled", *profile.StripeAccountStatus)
	})
}