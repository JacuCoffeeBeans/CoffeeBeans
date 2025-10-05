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

	t.Run("Success with valid signature", func(t *testing.T) {
		payload := `{"id": "evt_test", "type": "payment_intent.succeeded", "data": {"object": {}}}`
		req := createTestRequest(t, payload, testWebhookSecret)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(handleStripeWebhook)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("Fail with no signature header", func(t *testing.T) {
		payload := `{"id": "evt_test", "type": "payment_intent.succeeded", "data": {"object": {}}}`
		req := httptest.NewRequest("POST", "/api/webhooks/stripe", strings.NewReader(payload))

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(handleStripeWebhook)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Fail with invalid signature", func(t *testing.T) {
		payload := `{"id": "evt_test", "type": "payment_intent.succeeded", "data": {"object": {}}}`
		req := createTestRequest(t, payload, "wrong_secret")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(handleStripeWebhook)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
