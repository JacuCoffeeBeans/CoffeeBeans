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

	var beans []Bean
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
