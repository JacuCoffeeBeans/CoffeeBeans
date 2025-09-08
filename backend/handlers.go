package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
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
	userID := r.Context().Value(userIDKey).(string)

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

// updateBeanHandler は既存のコーヒー豆のデータを更新します
func (a *Api) updateBeanHandler(w http.ResponseWriter, r *http.Request) {
	// contextから、認証ミドルウェアが設定したユーザーIDを取得
	userID, ok := r.Context().Value(userIDKey).(string)
	if !ok || strings.TrimSpace(userID) == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// URLからIDを取得
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid bean ID", http.StatusBadRequest)
		return
	}

	var bean Bean
	if err := json.NewDecoder(r.Body).Decode(&bean); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Store（DB）のBeanを更新する
	updatedBean, err := a.store.UpdateBean(r.Context(), id, userID, &bean)
	if err != nil {
		// pgx.ErrNoRowsは、更新対象が見つからなかった（IDが違うか、所有者でない）場合に返される
		if err.Error() == "no rows in result set" {
			// 他のユーザーの所有物である可能性を示唆しないよう、一般的なNot Foundを返す
			http.Error(w, "Bean not found or you don't have permission to update it", http.StatusNotFound)
			return
		}
		log.Printf("ERROR: Failed to update bean in DB: %v", err)
		http.Error(w, "Failed to update bean", http.StatusInternalServerError)
		return
	}

	// 成功したら、更新後のデータを返す
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(updatedBean); err != nil {
		log.Printf("ERROR: Failed to encode updated bean to JSON: %v", err)
	}
}

// deleteBeanHandler は既存のコーヒー豆のデータを削除します
func (a *Api) deleteBeanHandler(w http.ResponseWriter, r *http.Request) {
	// contextから、認証ミドルウェアが設定したユーザーIDを取得
	userID, ok := r.Context().Value(userIDKey).(string)
	if !ok || strings.TrimSpace(userID) == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// URLからIDを取得
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid bean ID", http.StatusBadRequest)
		return
	}

	// Store（DB）のBeanを削除する
	err = a.store.DeleteBean(r.Context(), id, userID)
	if err != nil {
		// pgx.ErrNoRowsは、削除対象が見つからなかった（IDが違うか、所有者でない）場合に返される
		if err.Error() == "no rows in result set" {
			// 他のユーザーの所有物である可能性を示唆しないよう、一般的なNot Foundを返す
			http.Error(w, "Bean not found or you don't have permission to delete it", http.StatusNotFound)
			return
		}
		log.Printf("ERROR: Failed to delete bean from DB: %v", err)
		http.Error(w, "Failed to delete bean", http.StatusInternalServerError)
		return
	}

	// 成功したら、ステータスコード204を返す
	w.WriteHeader(http.StatusNoContent)
}

// getMyBeansHandler は認証されているユーザー自身の豆リストを取得します
func (a *Api) getMyBeansHandler(w http.ResponseWriter, r *http.Request) {
	// contextから、認証ミドルウェアが設定したユーザーIDを取得
	userID, ok := r.Context().Value(userIDKey).(string)
	if !ok || strings.TrimSpace(userID) == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	beans, err := a.store.GetBeansByUserID(r.Context(), userID)
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

// beansHandlerは "/api/beans" へのリクエストをHTTPメソッドによって振り分ける
// *GETの場合は認証を要求しない
// *POSTの場合は認証を要求する
func (a *Api) beansHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.getBeansHandler(w, r)
	case http.MethodPost:
		// POSTの場合は、contextにミドルウェアで認証済みのuserIDが入っているかチェック
		userID, ok := r.Context().Value(userIDKey).(string)
		if !ok || strings.TrimSpace(userID) == "" {
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

	case http.MethodPut:
		// PUT（更新）の場合は、認証済みユーザーである必要があるので、ここでチェック
		userID, ok := r.Context().Value(userIDKey).(string)
		if !ok || strings.TrimSpace(userID) == "" {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		a.updateBeanHandler(w, r)

	case http.MethodDelete:
		// DELETE（削除）の場合も、認証済みユーザーである必要があるので、ここでチェック
		userID, ok := r.Context().Value(userIDKey).(string)
		if !ok || strings.TrimSpace(userID) == "" {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		a.deleteBeanHandler(w, r)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// addCartItemHandler はカートに商品を追加します
func (a *Api) addCartItemHandler(w http.ResponseWriter, r *http.Request) {
	// contextから、認証ミドルウェアが設定したユーザーIDを取得
	userID, ok := r.Context().Value(userIDKey).(string)
	if !ok || strings.TrimSpace(userID) == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req AddCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 簡単なバリデーション
	if req.BeanID <= 0 || req.Quantity <= 0 {
		http.Error(w, "BeanID and quantity must be positive", http.StatusBadRequest)
		return
	}

	// Store（DB）にカートアイテムを追加/更新
	cartItem, err := a.store.AddOrUpdateCartItem(r.Context(), userID, req)
	if err != nil {
		log.Printf("ERROR: Failed to add or update cart item: %v", err)
		// ここで、例えば "foreign key constraint" のような具体的なDBエラーをチェックして、
		// 存在しないbean_idが指定された場合に404 Not Foundを返すことも可能
		http.Error(w, "Failed to process cart operation", http.StatusInternalServerError)
		return
	}

	// 成功したら、ステータスコード200と登録したデータを返す
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(cartItem); err != nil {
		log.Printf("ERROR: Failed to encode cart item to JSON: %v", err)
	}
}