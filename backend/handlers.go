package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/account"
	"github.com/stripe/stripe-go/v72/accountlink"
	"github.com/stripe/stripe-go/v72/paymentintent"
	"github.com/stripe/stripe-go/v72/webhook"
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

// getCartHandler は認証されているユーザーのカートの中身を取得します
func (a *Api) getCartHandler(w http.ResponseWriter, r *http.Request) {
	// contextから、認証ミドルウェアが設定したユーザーIDを取得
	userID, ok := r.Context().Value(userIDKey).(string)
	if !ok || strings.TrimSpace(userID) == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Storeからカートの中身を取得
	cartItems, err := a.store.GetCartItemsByUserID(r.Context(), userID)
	if err != nil {
		log.Printf("ERROR: Failed to get cart items from DB: %v", err)
		http.Error(w, "Failed to get cart items", http.StatusInternalServerError)
		return
	}

	// cartItemsがnilの場合（DBからは起こりにくいが）、空のスライスを返す
	if cartItems == nil {
		cartItems = []CartItemDetail{}
	}

	// 成功したら、カートの中身をJSONで返す
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(cartItems); err != nil {
		log.Printf("ERROR: Failed to encode cart items to JSON: %v", err)
	}
}

// cartItemDetailHandlerは /api/cart/items/{id} へのリクエストをHTTPメソッドによって振り分ける
func (a *Api) cartItemDetailHandler(w http.ResponseWriter, r *http.Request) {
	// 認証済みユーザーである必要があるので、ここでチェック
	userID, ok := r.Context().Value(userIDKey).(string)
	if !ok || strings.TrimSpace(userID) == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodPut:
		a.updateCartItemHandler(w, r)
	case http.MethodDelete:
		a.deleteCartItemHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// updateCartItemHandler はカート内の商品の数量を更新します
func (a *Api) updateCartItemHandler(w http.ResponseWriter, r *http.Request) {
	// contextから、認証ミドルウェアが設定したユーザーIDを取得
	userID := r.Context().Value(userIDKey).(string)

	// URLからIDを取得
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "Cart item ID is required", http.StatusBadRequest)
		return
	}

	var req UpdateCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 数量は1以上であるべき
	if req.Quantity <= 0 {
		http.Error(w, "Quantity must be positive", http.StatusBadRequest)
		return
	}

	// Store（DB）のカートアイテムを更新する
	updatedItem, err := a.store.UpdateCartItemQuantity(r.Context(), idStr, userID, req.Quantity)
	if err != nil {
		// pgx.ErrNoRowsは、更新対象が見つからなかった（IDが違うか、所有者でない）場合に返される
		if err.Error() == "no rows in result set" {
			http.Error(w, "Cart item not found or you don't have permission to update it", http.StatusNotFound)
			return
		}
		log.Printf("ERROR: Failed to update cart item in DB: %v", err)
		http.Error(w, "Failed to update cart item", http.StatusInternalServerError)
		return
	}

	// 成功したら、更新後のデータを返す
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(updatedItem); err != nil {
		log.Printf("ERROR: Failed to encode updated cart item to JSON: %v", err)
	}
}

// deleteCartItemHandler はカートから商品を削除します
func (a *Api) deleteCartItemHandler(w http.ResponseWriter, r *http.Request) {
	// contextから、認証ミドルウェアが設定したユーザーIDを取得
	userID := r.Context().Value(userIDKey).(string)

	// URLからIDを取得
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "Cart item ID is required", http.StatusBadRequest)
		return
	}

	// Store（DB）のカートアイテムを削除する
	err := a.store.DeleteCartItem(r.Context(), idStr, userID)
	if err != nil {
		// pgx.ErrNoRowsは、削除対象が見つからなかった（IDが違うか、所有者でない）場合に返される
		if err.Error() == "no rows in result set" {
			http.Error(w, "Cart item not found or you don't have permission to delete it", http.StatusNotFound)
			return
		}
		log.Printf("ERROR: Failed to delete cart item from DB: %v", err)
		http.Error(w, "Failed to delete cart item", http.StatusInternalServerError)
		return
	}

	// 成功したら、ステータスコード204を返す
	w.WriteHeader(http.StatusNoContent)
}

// profileHandlerは "/api/profile" へのリクエストをHTTPメソッドによって振り分ける
func (a *Api) profileHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		a.createProfileHandler(w, r)
	case http.MethodPut:
		a.updateProfileHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// createProfileHandler は新しいプロフィールを登録します
func (a *Api) createProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(string)

	var profile Profile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	profile.UserID = userID

	newProfile, err := a.store.CreateProfile(r.Context(), &profile)
	if err != nil {
		log.Printf("ERROR: Failed to create profile in DB: %v", err)
		http.Error(w, "Failed to create profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newProfile); err != nil {
		log.Printf("ERROR: Failed to encode new profile to JSON: %v", err)
	}
}

// updateProfileHandler は既存のプロフィールを更新します
func (a *Api) updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(string)

	var profile Profile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	profile.UserID = userID

	updatedProfile, err := a.store.UpdateProfile(r.Context(), &profile)
	if err != nil {
		log.Printf("ERROR: Failed to update profile in DB: %v", err)
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(updatedProfile); err != nil {
		log.Printf("ERROR: Failed to encode updated profile to JSON: %v", err)
	}
}

// createPaymentIntentHandler はStripeのPaymentIntentを作成し、client_secretを返します
func (a *Api) createPaymentIntentHandler(w http.ResponseWriter, r *http.Request) {
	// 認証済みユーザーでなければエラー
	userID, ok := r.Context().Value(userIDKey).(string)
	if !ok || strings.TrimSpace(userID) == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// POSTメソッドでなければエラー
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// ユーザーのカート情報をDBから取得
	cartItems, err := a.store.GetCartItemsByUserID(r.Context(), userID)
	if err != nil {
		log.Printf("ERROR: Failed to get cart items from DB: %v", err)
		http.Error(w, "Failed to get cart items", http.StatusInternalServerError)
		return
	}

	// カートが空の場合はエラー
	if len(cartItems) == 0 {
		http.Error(w, "Cart is empty", http.StatusBadRequest)
		return
	}

	// 合計金額を計算
	var totalAmount int64
	for _, item := range cartItems {
		totalAmount += int64(item.Price) * int64(item.Quantity)
	}

	// StripeのAPIキーを設定
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	// PaymentIntentを作成
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(totalAmount),
		Currency: stripe.String(string(stripe.CurrencyJPY)), // 通貨をJPYに設定
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}
	params.AddMetadata("user_id", userID)

	pi, err := paymentintent.New(params)
	if err != nil {
		log.Printf("ERROR: Failed to create PaymentIntent: %v", err)
		http.Error(w, "Failed to create PaymentIntent", http.StatusInternalServerError)
		return
	}

	// レスポンスを作成
	response := map[string]string{"client_secret": pi.ClientSecret}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("ERROR: Failed to encode response to JSON: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleStripeWebhook はStripeからのWebhookを受け取り処理します
func (a *Api) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("ERROR: Failed to read webhook body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusServiceUnavailable)
		return
	}

	signatureHeader := r.Header.Get("Stripe-Signature")
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")

	event, err := webhook.ConstructEvent(payload, signatureHeader, webhookSecret)
	if err != nil {
		log.Printf("ERROR: Webhook signature verification failed: %v", err)
		http.Error(w, "Webhook signature verification failed", http.StatusBadRequest)
		return
	}

	switch event.Type {
	case "payment_intent.succeeded":
		var paymentIntent stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
			log.Printf("ERROR: Failed to unmarshal payment_intent.succeeded: %v", err)
			http.Error(w, "Failed to parse webhook data", http.StatusBadRequest)
			return
		}
		log.Printf("✅ PaymentIntent succeeded: %s", paymentIntent.ID)

		userID, ok := paymentIntent.Metadata["user_id"]
		shortUserID := userID
		if len(userID) > 8 {
			shortUserID = userID[:8]
		}
		if !ok || userID == "" {
			log.Printf("ERROR: user_id not found in payment intent metadata for pi_id: %s", paymentIntent.ID)
			http.Error(w, "User ID not found in metadata", http.StatusBadRequest)
			return
		}

		cartItems, err := a.store.GetCartItemsByUserID(r.Context(), userID)
		if err != nil {
			log.Printf("ERROR: Failed to get cart items for user %s: %v", shortUserID, err)
			http.Error(w, "Failed to get cart items", http.StatusInternalServerError)
			return
		}
		if len(cartItems) == 0 {
			log.Printf("INFO: Cart is empty for user %s on payment success, possibly already processed.", shortUserID)
			w.WriteHeader(http.StatusOK)
			return
		}

		// トランザクションを開始
		tx, err := a.dbpool.Begin(r.Context())
		if err != nil {
			log.Printf("ERROR: Failed to begin transaction: %v", err)
			http.Error(w, "Failed to process order", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(r.Context()) // エラー発生時にロールバック

		storeWithTx := NewStore(tx)
		order := &Order{
			UserID:                userID,
			Status:                "succeeded",
			TotalAmount:           int(paymentIntent.Amount),
			Currency:              string(paymentIntent.Currency),
			PaymentMethodType:     paymentIntent.PaymentMethodTypes[0],
			StripePaymentIntentID: paymentIntent.ID,
		}

		// 注文を作成
		if _, err := storeWithTx.CreateOrder(r.Context(), order, cartItems); err != nil {
			log.Printf("ERROR: Failed to create order for user %s: %v", shortUserID, err)
			http.Error(w, "Failed to create order", http.StatusInternalServerError)
			return
		}

		// カートを空にする
		if err := storeWithTx.ClearCart(r.Context(), userID); err != nil {
			log.Printf("ERROR: Failed to clear cart for user %s: %v", shortUserID, err)
			http.Error(w, "Failed to clear cart", http.StatusInternalServerError)
			return
		}

		// トランザクションをコミット
		if err := tx.Commit(r.Context()); err != nil {
			log.Printf("ERROR: Failed to commit transaction for user %s: %v", shortUserID, err)
			http.Error(w, "Failed to finalize order", http.StatusInternalServerError)
			return
		}

		log.Printf("🎉 Order created successfully for user %s", shortUserID)

	case "payment_intent.payment_failed":
		var paymentIntent stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
			log.Printf("ERROR: Failed to unmarshal payment_intent.payment_failed: %v", err)
			http.Error(w, "Failed to parse webhook data", http.StatusBadRequest)
			return
		}
		log.Printf("❌ PaymentIntent failed: %s, Reason: %s", paymentIntent.ID, paymentIntent.LastPaymentError.Msg)

		userID, ok := paymentIntent.Metadata["user_id"]
		shortUserID := userID
		if len(userID) > 8 {
			shortUserID = userID[:8]
		}
		if !ok || userID == "" {
			log.Printf("ERROR: user_id not found in payment intent metadata for pi_id: %s", paymentIntent.ID)
			http.Error(w, "User ID not found in metadata", http.StatusBadRequest)
			return
		}

		cartItems, err := a.store.GetCartItemsByUserID(r.Context(), userID)
		if err != nil {
			log.Printf("ERROR: Failed to get cart items for user %s: %v", shortUserID, err)
			http.Error(w, "Failed to get cart items", http.StatusInternalServerError)
			return
		}

		order := &Order{
			UserID:                userID,
			Status:                "failed",
			TotalAmount:           int(paymentIntent.Amount),
			Currency:              string(paymentIntent.Currency),
			PaymentMethodType:     paymentIntent.PaymentMethodTypes[0],
			StripePaymentIntentID: paymentIntent.ID,
		}

		// 失敗した注文も記録する
		if _, err := a.store.CreateOrder(r.Context(), order, cartItems); err != nil {
			log.Printf("ERROR: Failed to create failed order record for user %s: %v", shortUserID, err)
			http.Error(w, "Failed to create order record", http.StatusInternalServerError)
			return
		}
		log.Printf("📝 Failed order recorded for user %s", shortUserID)

	default:
		log.Printf("🤷‍♀️ Unhandled event type: %s", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}

// createStripeAccountLinkHandler はStripe Connectアカウントを作成し、オンボーディング用のURLを返します
func (a *Api) createStripeAccountLinkHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(string)

	// Stripe APIキーを設定
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	// ユーザーのプロフィールを取得
	profile, err := a.store.GetProfileByUserID(r.Context(), userID)
	if err != nil {
		log.Printf("ERROR: Failed to get profile for user %s: %v", userID, err)
		http.Error(w, "Failed to get user profile", http.StatusInternalServerError)
		return
	}

	// 既にStripeアカウントIDを持っているか確認
	var accountID string
	if profile.StripeAccountID != nil && *profile.StripeAccountID != "" {
		accountID = *profile.StripeAccountID
	} else {
		// Stripe Connectアカウントを作成
		params := &stripe.AccountParams{
			Type:    stripe.String(string(stripe.AccountTypeExpress)),
			Country: stripe.String("JP"),
			Email:   stripe.String(profile.DisplayName + "@example.com"), // 仮のメールアドレス
			Capabilities: &stripe.AccountCapabilitiesParams{
				CardPayments: &stripe.AccountCapabilitiesCardPaymentsParams{Requested: stripe.Bool(true)},
				Transfers:    &stripe.AccountCapabilitiesTransfersParams{Requested: stripe.Bool(true)},
			},
		}
		newAccount, err := account.New(params)
		if err != nil {
			log.Printf("ERROR: Failed to create Stripe account for user %s: %v", userID, err)
			http.Error(w, "Failed to create Stripe account", http.StatusInternalServerError)
			return
		}
		accountID = newAccount.ID

		// プロフィールを更新
		if err := a.store.UpdateStripeAccount(r.Context(), userID, accountID, strconv.FormatBool(newAccount.ChargesEnabled)); err != nil {
			log.Printf("ERROR: Failed to update profile with Stripe account ID for user %s: %v", userID, err)
			http.Error(w, "Failed to update profile", http.StatusInternalServerError)
			return
		}
	}

	// アカウントリンクを作成
	linkParams := &stripe.AccountLinkParams{
		Account:    stripe.String(accountID),
		RefreshURL: stripe.String("http://localhost:8080/api/stripe/connect/refresh"),      // 再度このハンドラを呼び出すURL
		ReturnURL:  stripe.String("http://localhost:5173/my/beans?stripe_connect=success"), // オンボーディング完了後のリダイレクト先
		Type:       stripe.String("account_onboarding"),
	}
	accountLink, err := accountlink.New(linkParams)
	if err != nil {
		log.Printf("ERROR: Failed to create account link for user %s: %v", userID, err)
		http.Error(w, "Failed to create account link", http.StatusInternalServerError)
		return
	}

	// レスポンスを返す
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"url": accountLink.URL}); err != nil {
		log.Printf("ERROR: Failed to encode response to JSON: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleStripeConnectRedirectHandler はStripeからのリダイレクトを処理します
func (a *Api) handleStripeConnectRedirectHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(string)

	// ユーザーのプロフィールを取得
	profile, err := a.store.GetProfileByUserID(r.Context(), userID)
	if err != nil {
		log.Printf("ERROR: Failed to get profile for user %s: %v", userID, err)
		http.Error(w, "User profile not found", http.StatusNotFound)
		return
	}

	if profile.StripeAccountID == nil || *profile.StripeAccountID == "" {
		log.Printf("ERROR: Stripe account ID not found for user %s", userID)
		http.Error(w, "Stripe account not set up", http.StatusBadRequest)
		return
	}

	// Stripeアカウント情報を取得
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	stripeAccount, err := account.GetByID(*profile.StripeAccountID, nil)
	if err != nil {
		log.Printf("ERROR: Failed to retrieve Stripe account for user %s: %v", userID, err)
		http.Error(w, "Failed to retrieve Stripe account", http.StatusInternalServerError)
		return
	}

	// アカウントステータスを更新
	status := "restricted"
	if stripeAccount.ChargesEnabled {
		status = "enabled"
	}
	if err := a.store.UpdateStripeAccount(r.Context(), userID, *profile.StripeAccountID, status); err != nil {
		log.Printf("ERROR: Failed to update Stripe account status for user %s: %v", userID, err)
		http.Error(w, "Failed to update account status", http.StatusInternalServerError)
		return
	}

	// フロントエンドの特定ページにリダイレクト
	http.Redirect(w, r, "http://localhost:5173/my/beans?stripe_connect=success", http.StatusFound)
}
