// backend/main_test.go

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect" // データを深く比較するために追加
	"testing"
)

// TestGetBeansHandler は getBeansHandler のテスト関数です
func TestGetBeansHandler(t *testing.T) {
	// --- 準備 (Arrange) ---

	// テスト用のHTTPリクエストを作成
	// GETメソッドで "/api/beans" を呼び出すリクエスト
	req, err := http.NewRequest("GET", "/api/beans", nil)
	if err != nil {
		t.Fatalf("リクエストの作成に失敗しました: %v", err)
	}

	// レスポンスを記録するためのレコーダーを作成
	// これが、テスト中の「偽のブラウザ」の役割をします
	rr := httptest.NewRecorder()

	// テスト対象のハンドラを準備
	handler := http.HandlerFunc(getBeansHandler)

	// --- 実行 (Act) ---

	// 作成したリクエストをハンドラに渡し、レスポンスをレコーダーに記録
	handler.ServeHTTP(rr, req)

	// --- 検証 (Assert) ---

	// 1. ステータスコードが200 OKか？
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("期待と異なるステータスコードです: got %v want %v", status, http.StatusOK)
	}

	// 2. レスポンスボディのJSONを、[]Bean のスライスに変換（デコード）してみる
	var actualBeans []Bean
	if err := json.NewDecoder(rr.Body).Decode(&actualBeans); err != nil {
		t.Fatalf("レスポンスボディのJSONデコードに失敗しました: %v", err)
	}

	// 3. 返ってきたデータの中身は、私たちが定義したダミーデータと一致するか？
	// reflect.DeepEqual は、スライスや構造体など複雑なデータでも、中身が完全に一致するかを比較してくれます
	if !reflect.DeepEqual(actualBeans, beans) {
		t.Errorf("期待と異なるレスポンスボディです: got %v want %v", actualBeans, beans)
	}
}