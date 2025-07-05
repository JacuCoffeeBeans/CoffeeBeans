// backend/main_test.go

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestGetBeansHandler(t *testing.T) {
	// --- 準備 (Arrange) ---

	// テスト用のHTTPリクエストを作成
	// GETメソッドで "/api/beans" を呼び出すリクエスト
	req, err := http.NewRequest("GET", "/api/beans", nil)
	if err != nil {
		t.Fatalf("リクエストの作成に失敗しました: %v", err)
	}

	// レスポンスを記録するためのレコーダーを作成
	// これが、テスト中の「偽のブラウザ」の役割
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

	// 2. レスポンスボディのJSONを、[]Bean のスライスに変換（デコード）できるか？
	var actualBeans []Bean
	if err := json.NewDecoder(rr.Body).Decode(&actualBeans); err != nil {
		t.Fatalf("レスポンスボディのJSONデコードに失敗しました: %v", err)
	}

	// 3. 返ってきたデータの中身は、定義したダミーデータと一致するか？
	if !reflect.DeepEqual(actualBeans, beans) {
		t.Errorf("期待と異なるレスポンスボディです: got %v want %v", actualBeans, beans)
	}
}

func TestGetBeanHandler(t *testing.T) {
	// テストケースをまとめて定義
	testCases := []struct {
		name           string // テスト名
		id             string // テストで使うID
		expectedStatus int    // 期待するHTTPステータス
		expectedName   string // 期待する豆の名前 (成功時のみ)
	}{
		{
			name:           "正常系: 存在するID",
			id:             "1",
			expectedStatus: http.StatusOK,
			expectedName:   "エチオピア イルガチェフェ",
		},
		{
			name:           "異常系: 存在しないID",
			id:             "99",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "異常系: 不正なID (文字列)",
			id:             "abc",
			expectedStatus: http.StatusBadRequest,
		},
	}

	// 各テストケースを順番に実行
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// --- 準備 ---
			req, err := http.NewRequest("GET", "/api/beans/"+tc.id, nil)
			if err != nil {
				t.Fatalf("リクエストの作成に失敗しました: %v", err)
			}
			// PathValueをテストで使えるように、リクエストに値をセット
			req.SetPathValue("id", tc.id)

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(getBeanHandler)

			// --- 実行 ---
			handler.ServeHTTP(rr, req)

			// --- 検証 ---
			if status := rr.Code; status != tc.expectedStatus {
				t.Errorf("期待と異なるステータスコードです: got %v want %v", status, tc.expectedStatus)
			}

			// 成功時のみ、ボディの中身を検証
			if tc.expectedStatus == http.StatusOK {
				var bean Bean
				if err := json.NewDecoder(rr.Body).Decode(&bean); err != nil {
					t.Fatalf("レスポンスボディのJSONデコードに失敗しました: %v", err)
				}
				if bean.Name != tc.expectedName {
					t.Errorf("期待と異なるレスポンスボディです: got %v want %v", bean.Name, tc.expectedName)
				}
			}
		})
	}
}

func TestFindBeanByID(t *testing.T) {
	// テストケース1: 存在するIDを検索
	t.Run("存在するIDの場合", func(t *testing.T) {
		id := 2
		bean, found := findBeanByID(id)

		if !found {
			t.Errorf("見つかるはずの豆が見つかりませんでした (ID: %d)", id)
		}
		if bean == nil {
			t.Fatal("beanがnilであってはいけません")
		}
		if bean.ID != id {
			t.Errorf("期待と異なるIDの豆が見つかりました: got %d want %d", bean.ID, id)
		}
		if bean.Name != "ブラジル サントスNo.2" {
			t.Errorf("期待と異なる名前の豆が見つかりました: got %s", bean.Name)
		}
	})

	// テストケース2: 存在しないIDを検索
	t.Run("存在しないIDの場合", func(t *testing.T) {
		id := 999
		bean, found := findBeanByID(id)

		if found {
			t.Errorf("見つからないはずの豆が見つかりました (ID: %d)", id)
		}
		if bean != nil {
			t.Errorf("beanがnilであるべきです: got %v", bean)
		}
	})
}
