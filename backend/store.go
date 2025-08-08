package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

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
}

// Store はデータベース接続プールを保持します
type Store struct {
	db *pgxpool.Pool
}

// NewStore は新しいStoreインスタンスを作成します
func NewStore(db *pgxpool.Pool) *Store {
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
