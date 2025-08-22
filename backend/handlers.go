package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

// getBeansHandler はStoreを使ってDBから全件取得する
func (a *Api) getBeansHandler(w http.ResponseWriter, r *http.Request) {
	beans, err := a.store.GetAllBeans(r.Context())
	if err != nil {
		log.Printf("ERROR: Failed to get beans from DB: %v", err)
		http.Error(w, "Failed to get beans from DB", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(beans); err != nil {
		http.Error(w, "Failed to encode beans to JSON", http.StatusInternalServerError)
	}
}

// getBeanHandler はStoreを使ってDBから1件取得する
func (a *Api) getBeanHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid bean ID", http.StatusBadRequest)
		return
	}

	bean, err := a.store.GetBeanByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to get bean from DB", http.StatusInternalServerError)
		return
	}
	if bean == nil { // Storeからデータなし・エラーなしで返ってきた場合はNot Found
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(bean); err != nil {
		http.Error(w, "Failed to encode bean to JSON", http.StatusInternalServerError)
	}
}

// healthCheckHandler はルートURLのハンドラです
func (a *Api) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Backend server is running!")
}

// createBeanHandler は新しいコーヒー豆のデータを登録します
func (a *Api) createBeanHandler(w http.ResponseWriter, r *http.Request) {
	// contextから、認証ミドルウェアが設定したユーザーIDを取得
	userID := r.Context().Value("userID").(string)

	var bean Bean
	if err := json.NewDecoder(r.Body).Decode(&bean); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 必須フィールドの簡単なバリデーション
	if bean.Name == "" || bean.Origin == "" {
		http.Error(w, "Name and origin are required", http.StatusBadRequest)
		return
	}

	bean.UserID = userID

	// Store（DB）に新しいBeanを登録する
	newBean, err := a.store.CreateBean(r.Context(), &bean)
	if err != nil {
		log.Printf("ERROR: Failed to create bean in DB: %v", err)
		http.Error(w, "Failed to create bean", http.StatusInternalServerError)
		return
	}

	// 成功したら、ステータスコード201と登録したデータを返す
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newBean); err != nil {
		// ここでのエラーはクライアントへのレスポンス送信の失敗
		log.Printf("ERROR: Failed to encode new bean to JSON: %v", err)
	}
}

// beansHandlerは "/api/beans" へのリクエストをHTTPメソッドによって振り分ける
// *GETの場合は認証を要求しない
// *POSTの場合は認証を要求する
func (a *Api) beansHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.getBeansHandler(w, r)
	case http.MethodPost:
		// POSTの場合は、contextにミドルウェアで認証済みのuserIDが入っているかチェック
		if r.Context().Value("userID") == nil {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		a.createBeanHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// beanDetailHandlerは /api/beans/{id} へのリクエストをHTTPメソッドによって振り分ける
// *GETの場合は認証を要求しない
func (a *Api) beanDetailHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// GET（詳細取得）は認証不要なので、そのままハンドラを呼ぶ
		a.getBeanHandler(w, r)

	// case http.MethodPut:
	// 将来、更新機能をここに実装する

	// case http.MethodDelete:
	// 将来、削除機能をここに実装する

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
