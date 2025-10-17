package main

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Querierインターフェースは、pgxpool.Poolとpgx.Txの両方が満たすメソッドを定義します。
// これにより、通常の操作とトランザクション内の操作を同じコードで扱えるようになります。
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

// Bean 構造体
type Bean struct {
	ID           int       `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Name         string    `json:"name"`
	Origin       string    `json:"origin"`
	Price        int       `json:"price"`
	Process      string    `json:"process"`
	RoastProfile string    `json:"roast_profile"`
	UserID       string    `json:"user_id"`
}

// Store はデータベース接続またはトランザクションを保持します
type Store struct {
	db Querier
}

// NewStore は新しいStoreインスタンスを作成します
func NewStore(db Querier) *Store {
	return &Store{db: db}
}

// GetAllBeans はbeansテーブルから全ての豆を取得します
func (s *Store) GetAllBeans(ctx context.Context) ([]Bean, error) {
	// 標準的なSQLクエリを記述
	rows, err := s.db.Query(ctx, "SELECT id, created_at, updated_at, name, origin, price, process, roast_profile FROM beans ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	beans := []Bean{}
	for rows.Next() {
		var b Bean
		// 取得したデータをBean構造体にスキャン
		if err := rows.Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt, &b.Name, &b.Origin, &b.Price, &b.Process, &b.RoastProfile); err != nil {
			return nil, err
		}
		beans = append(beans, b)
	}
	return beans, nil
}

// GetBeanByID は指定されたIDの豆を1件取得します
func (s *Store) GetBeanByID(ctx context.Context, id int) (*Bean, error) {
	var b Bean
	err := s.db.QueryRow(ctx, "SELECT id, created_at, updated_at, name, origin, price, process, roast_profile FROM beans WHERE id = $1", id).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt, &b.Name, &b.Origin, &b.Price, &b.Process, &b.RoastProfile)
	if err != nil {
		// データが見つからない場合もエラーになるので、それをハンドリングする必要がある（今後の課題）
		return nil, err
	}
	return &b, nil
}

// CreateBean は新しいコーヒー豆のデータをDBに挿入します
func (s *Store) CreateBean(ctx context.Context, bean *Bean) (*Bean, error) {
	var newBean Bean
	// SQLクエリ: 新しいデータを挿入し、その結果（IDなど）を返す
	query := `INSERT INTO beans (name, origin, price, process, roast_profile, user_id, updated_at)
			   VALUES ($1, $2, $3, $4, $5, $6, NOW())
			   RETURNING id, created_at, updated_at, name, origin, price, process, roast_profile, user_id`

	err := s.db.QueryRow(ctx, query, bean.Name, bean.Origin, bean.Price, strings.ToLower(bean.Process), strings.ToLower(bean.RoastProfile), bean.UserID).Scan(
		&newBean.ID,
		&newBean.CreatedAt,
		&newBean.UpdatedAt,
		&newBean.Name,
		&newBean.Origin,
		&newBean.Price,
		&newBean.Process,
		&newBean.RoastProfile,
		&newBean.UserID,
	)

	if err != nil {
		return nil, err
	}

	return &newBean, nil
}

// UpdateBean は指定されたIDのコーヒー豆の情報を更新します
func (s *Store) UpdateBean(ctx context.Context, id int, userID string, bean *Bean) (*Bean, error) {
	var updatedBean Bean

	// SQLクエリ: 既存のデータを更新し、その結果を返す
	// WHERE句でidとuser_idの両方をチェックすることで、所有者のみが更新できるようにする
	query := `UPDATE beans
			   SET name = $1, origin = $2, price = $3, process = $4, roast_profile = $5, updated_at = NOW()
			   WHERE id = $6 AND user_id = $7
			   RETURNING id, created_at, updated_at, name, origin, price, process, roast_profile, user_id`

	err := s.db.QueryRow(ctx, query, bean.Name, bean.Origin, bean.Price, strings.ToLower(bean.Process), strings.ToLower(bean.RoastProfile), id, userID).Scan(
		&updatedBean.ID,
		&updatedBean.CreatedAt,
		&updatedBean.UpdatedAt,
		&updatedBean.Name,
		&updatedBean.Origin,
		&updatedBean.Price,
		&updatedBean.Process,
		&updatedBean.RoastProfile,
		&updatedBean.UserID,
	)

	if err != nil {
		// pgx.ErrNoRowsは、行が見つからなかった（つまり、IDが違うか、ユーザーが所有者でない）場合に返される
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows // エラーをラップせず、そのまま返す
		}
		return nil, err
	}

	return &updatedBean, nil
}

// DeleteBean は指定されたIDのコーヒー豆の情報を削除します
func (s *Store) DeleteBean(ctx context.Context, id int, userID string) error {
	// SQLクエリ: 既存のデータを削除する
	// WHERE句でidとuser_idの両方をチェックすることで、所有者のみが削除できるようにする
	query := `DELETE FROM beans WHERE id = $1 AND user_id = $2`

	ct, err := s.db.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}

	// Execで、1行も影響がなかった場合、それは対象が見つからなかったことを意味する
	// (IDが違うか、userIDが違う)
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// GetBeansByUserID は指定されたユーザーIDの豆を全件取得します
func (s *Store) GetBeansByUserID(ctx context.Context, userID string) ([]Bean, error) {
	rows, err := s.db.Query(ctx, "SELECT id, created_at, updated_at, name, origin, price, process, roast_profile, user_id FROM beans WHERE user_id = $1 ORDER BY id DESC", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	beans := []Bean{}
	for rows.Next() {
		var b Bean
		if err := rows.Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt, &b.Name, &b.Origin, &b.Price, &b.Process, &b.RoastProfile, &b.UserID); err != nil {
			return nil, err
		}
		beans = append(beans, b)
	}
	return beans, nil
}

// CartItem 構造体
type CartItem struct {
	ID        string    `json:"id"`
	CartID    string    `json:"cart_id"`
	BeanID    int       `json:"bean_id"`
	Quantity  int       `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AddCartItemRequest 構造体
type AddCartItemRequest struct {
	BeanID   int `json:"bean_id"`
	Quantity int `json:"quantity"`
}

// AddOrUpdateCartItem はカートに商品を追加または更新します
func (s *Store) AddOrUpdateCartItem(ctx context.Context, userID string, req AddCartItemRequest) (*CartItem, error) {
	// 1. ユーザーIDに基づいてカートを取得または作成
	var cartID string
	err := s.db.QueryRow(ctx, "SELECT id FROM carts WHERE user_id = $1", userID).Scan(&cartID)
	if err == pgx.ErrNoRows {
		// カートが存在しない場合は新規作成
		err = s.db.QueryRow(ctx, "INSERT INTO carts (user_id) VALUES ($1) RETURNING id", userID).Scan(&cartID)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	// 2. カート内に同じ商品が既に存在するか確認
	var existingItemID string
	var currentQuantity int
	err = s.db.QueryRow(ctx, "SELECT id, quantity FROM cart_items WHERE cart_id = $1 AND bean_id = $2", cartID, req.BeanID).Scan(&existingItemID, &currentQuantity)

	var resultItem CartItem
	if err == pgx.ErrNoRows {
		// 3a. 存在しない場合は新規追加
		query := `INSERT INTO cart_items (cart_id, bean_id, quantity) VALUES ($1, $2, $3)
				  RETURNING id, cart_id, bean_id, quantity, created_at, updated_at`
		err = s.db.QueryRow(ctx, query, cartID, req.BeanID, req.Quantity).Scan(
			&resultItem.ID, &resultItem.CartID, &resultItem.BeanID, &resultItem.Quantity, &resultItem.CreatedAt, &resultItem.UpdatedAt,
		)
	} else if err != nil {
		return nil, err
	} else {
		// 3b. 存在する場合は数量を更新
		newQuantity := currentQuantity + req.Quantity
		query := `UPDATE cart_items SET quantity = $1, updated_at = NOW() WHERE id = $2
				  RETURNING id, cart_id, bean_id, quantity, created_at, updated_at`
		err = s.db.QueryRow(ctx, query, newQuantity, existingItemID).Scan(
			&resultItem.ID, &resultItem.CartID, &resultItem.BeanID, &resultItem.Quantity, &resultItem.CreatedAt, &resultItem.UpdatedAt,
		)
	}

	if err != nil {
		return nil, err
	}

	return &resultItem, nil
}

// CartItemDetail 構造体は、カート内の商品の詳細情報を保持します
type CartItemDetail struct {
	ID           string `json:"id"` // cart_itemsテーブルのID
	BeanID       int    `json:"bean_id"`
	Name         string `json:"name"`
	Price        int    `json:"price"`
	Quantity     int    `json:"quantity"`
	Process      string `json:"process"`
	RoastProfile string `json:"roast_profile"`
	// 必要に応じて他のBeanのフィールドも追加
}

// GetCartItemsByUserID は、ユーザーのカートの中身を商品の詳細情報とともに取得します
func (s *Store) GetCartItemsByUserID(ctx context.Context, userID string) ([]CartItemDetail, error) {
	// 1. ユーザーIDからカートIDを取得
	var cartID string
	err := s.db.QueryRow(ctx, "SELECT id FROM carts WHERE user_id = $1", userID).Scan(&cartID)
	if err == pgx.ErrNoRows {
		// カートが存在しない場合は、空のカートとして扱う
		return []CartItemDetail{}, nil
	}
	if err != nil {
		return nil, err
	}

	// 2. カートIDを使って、cart_itemsとbeansをJOINして商品情報を取得
	query := `
		SELECT
			ci.id,
			ci.bean_id,
			b.name,
			b.price,
			ci.quantity,
			b.process,
			b.roast_profile
		FROM
			cart_items ci
		JOIN
			beans b ON ci.bean_id = b.id
		WHERE
			ci.cart_id = $1
		ORDER BY
			ci.created_at DESC;
	`
	rows, err := s.db.Query(ctx, query, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []CartItemDetail
	for rows.Next() {
		var item CartItemDetail
		if err := rows.Scan(&item.ID, &item.BeanID, &item.Name, &item.Price, &item.Quantity, &item.Process, &item.RoastProfile); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	// rows.Err()で、ループ中に発生したエラーを確認
	if err = rows.Err(); err != nil {
		return nil, err
	}

	// カートに商品がない場合、itemsは空のスライスになる
	return items, nil
}

// UpdateCartItemRequest 構造体
type UpdateCartItemRequest struct {
	Quantity int `json:"quantity"`
}

// UpdateCartItemQuantity はカート内の商品の数量を更新します。所有権もチェックします。
func (s *Store) UpdateCartItemQuantity(ctx context.Context, cartItemID string, userID string, quantity int) (*CartItem, error) {
	query := `
		UPDATE cart_items ci
		SET quantity = $1, updated_at = NOW()
		FROM carts c
		WHERE ci.id = $2
		  AND ci.cart_id = c.id
		  AND c.user_id = $3
		RETURNING ci.id, ci.cart_id, ci.bean_id, ci.quantity, ci.created_at, ci.updated_at;
	`
	var updatedItem CartItem
	err := s.db.QueryRow(ctx, query, quantity, cartItemID, userID).Scan(
		&updatedItem.ID,
		&updatedItem.CartID,
		&updatedItem.BeanID,
		&updatedItem.Quantity,
		&updatedItem.CreatedAt,
		&updatedItem.UpdatedAt,
	)

	if err != nil {
		// pgx.ErrNoRowsは、行が見つからなかった（つまり、IDが違うか、ユーザーが所有者でない）場合に返される
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}

	return &updatedItem, nil
}

// DeleteCartItem はカートから商品を削除します。所有権もチェックします。
func (s *Store) DeleteCartItem(ctx context.Context, cartItemID string, userID string) error {
	query := `
		DELETE FROM cart_items ci
		USING carts c
		WHERE ci.id = $1
		  AND ci.cart_id = c.id
		  AND c.user_id = $2;
	`
	ct, err := s.db.Exec(ctx, query, cartItemID, userID)
	if err != nil {
		return err
	}

	// Execで、1行も影響がなかった場合、それは対象が見つからなかったことを意味する
	// (IDが違うか、userIDが違う)
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// Order 構造体
type Order struct {
	ID                    int       `json:"id"`
	UserID                string    `json:"user_id"`
	Status                string    `json:"status"`
	TotalAmount           int       `json:"total_amount"`
	Currency              string    `json:"currency"`
	PaymentMethodType     string    `json:"payment_method_type"`
	StripePaymentIntentID string    `json:"stripe_payment_intent_id"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// OrderItem 構造体
type OrderItem struct {
	ID              int `json:"id"`
	OrderID         int `json:"order_id"`
	BeanID          int `json:"bean_id"`
	PriceAtPurchase int `json:"price_at_purchase"`
	Quantity        int `json:"quantity"`
}

// CreateOrder は新しい注文をDBに作成します
func (s *Store) CreateOrder(ctx context.Context, order *Order, items []CartItemDetail) (*Order, error) {
	// 1. ordersテーブルに注文を挿入
	orderQuery := `
		INSERT INTO orders (user_id, status, total_amount, currency, payment_method_type, stripe_payment_intent_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	err := s.db.QueryRow(ctx, orderQuery, order.UserID, order.Status, order.TotalAmount, order.Currency, order.PaymentMethodType, order.StripePaymentIntentID).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// 2. order_itemsテーブルに注文商品を挿入
	batch := &pgx.Batch{}
	itemQuery := `
		INSERT INTO order_items (order_id, bean_id, price_at_purchase, quantity)
		VALUES ($1, $2, $3, $4)
	`
	for _, item := range items {
		batch.Queue(itemQuery, order.ID, item.BeanID, item.Price, item.Quantity)
	}

	br := s.db.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(items); i++ {
		_, err := br.Exec()
		if err != nil {
			return nil, err
		}
	}

	return order, nil
}

// ClearCart はユーザーのカートを空にします
func (s *Store) ClearCart(ctx context.Context, userID string) error {
	// ユーザーIDに紐づくカートIDを取得
	var cartID string
	err := s.db.QueryRow(ctx, "SELECT id FROM carts WHERE user_id = $1", userID).Scan(&cartID)
	if err != nil {
		if err == pgx.ErrNoRows {
			// カートが存在しない場合は、何もせず正常終了
			return nil
		}
		return err
	}

	// カートIDに紐づくすべてのカートアイテムを削除
	_, err = s.db.Exec(ctx, "DELETE FROM cart_items WHERE cart_id = $1", cartID)
	return err
}

// GetOrderByPaymentIntentID はStripeのPaymentIntent IDで注文を取得します
func (s *Store) GetOrderByPaymentIntentID(ctx context.Context, paymentIntentID string) (*Order, error) {
	var order Order
	query := `SELECT id, user_id, status, total_amount, currency, payment_method_type, stripe_payment_intent_id, created_at, updated_at FROM orders WHERE stripe_payment_intent_id = $1`
	err := s.db.QueryRow(ctx, query, paymentIntentID).Scan(
		&order.ID, &order.UserID, &order.Status, &order.TotalAmount, &order.Currency, &order.PaymentMethodType, &order.StripePaymentIntentID, &order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// Profile 構造体
type Profile struct {
	UserID           string    `json:"user_id"`
	DisplayName      string    `json:"display_name"`
	IconURL          string    `json:"icon_url"`
	PostCode         string    `json:"post_code"`
	Address          string    `json:"address"`
	AboutMe          string    `json:"about_me"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	StripeCustomerID *string    `json:"stripe_customer_id"`
	StripeAccountID *string `json:"stripe_account_id"`
	StripeAccountStatus *string `json:"stripe_account_status"`
}

// CreateProfile は新しいプロフィールをDBに挿入します
func (s *Store) CreateProfile(ctx context.Context, profile *Profile) (*Profile, error) {
	var newProfile Profile
	query := `INSERT INTO profiles (user_id, display_name, icon_url, post_code, address, about_me)
			   VALUES ($1, $2, $3, $4, $5, $6)
			   RETURNING user_id, display_name, icon_url, post_code, address, about_me, created_at, updated_at, stripe_customer_id, stripe_account_id, stripe_account_status`

	err := s.db.QueryRow(ctx, query, profile.UserID, profile.DisplayName, profile.IconURL, profile.PostCode, profile.Address, profile.AboutMe).Scan(
		&newProfile.UserID,
		&newProfile.DisplayName,
		&newProfile.IconURL,
		&newProfile.PostCode,
		&newProfile.Address,
		&newProfile.AboutMe,
		&newProfile.CreatedAt,
		&newProfile.UpdatedAt,
		&newProfile.StripeCustomerID,
		&newProfile.StripeAccountID,
		&newProfile.StripeAccountStatus,
	)

	if err != nil {
		return nil, err
	}

	return &newProfile, nil
}

// UpdateProfile は既存のプロフィールを更新します
func (s *Store) UpdateProfile(ctx context.Context, profile *Profile) (*Profile, error) {
	var updatedProfile Profile
	query := `UPDATE profiles
			   SET display_name = $1, icon_url = $2, post_code = $3, address = $4, about_me = $5, updated_at = NOW()
			   WHERE user_id = $6
			   RETURNING user_id, display_name, icon_url, post_code, address, about_me, created_at, updated_at, stripe_customer_id, stripe_account_id, stripe_account_status`

	err := s.db.QueryRow(ctx, query, profile.DisplayName, profile.IconURL, profile.PostCode, profile.Address, profile.AboutMe, profile.UserID).Scan(
		&updatedProfile.UserID,
		&updatedProfile.DisplayName,
		&updatedProfile.IconURL,
		&updatedProfile.PostCode,
		&updatedProfile.Address,
		&updatedProfile.AboutMe,
		&updatedProfile.CreatedAt,
		&updatedProfile.UpdatedAt,
		&updatedProfile.StripeCustomerID,
		&updatedProfile.StripeAccountID,
		&updatedProfile.StripeAccountStatus,
	)

	if err != nil {
		return nil, err
	}

	return &updatedProfile, nil
}

// GetProfileByUserID はユーザーIDでプロフィールを取得します
func (s *Store) GetProfileByUserID(ctx context.Context, userID string) (*Profile, error) {
	var profile Profile
	query := `SELECT user_id, display_name, icon_url, post_code, address, about_me, created_at, updated_at, stripe_customer_id, stripe_account_id, stripe_account_status FROM profiles WHERE user_id = $1`
	err := s.db.QueryRow(ctx, query, userID).Scan(
		&profile.UserID,
		&profile.DisplayName,
		&profile.IconURL,
		&profile.PostCode,
		&profile.Address,
		&profile.AboutMe,
		&profile.CreatedAt,
		&profile.UpdatedAt,
		&profile.StripeCustomerID,
		&profile.StripeAccountID,
		&profile.StripeAccountStatus,
	)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// UpdateStripeAccount はStripeアカウントIDとステータスを更新します
func (s *Store) UpdateStripeAccount(ctx context.Context, userID, accountID, status string) error {
	query := `UPDATE profiles SET stripe_account_id = $1, stripe_account_status = $2, updated_at = NOW() WHERE user_id = $3`
	_, err := s.db.Exec(ctx, query, accountID, status, userID)
	return err
}
