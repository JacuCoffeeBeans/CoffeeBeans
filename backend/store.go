package main

import postgrest "github.com/supabase-community/postgrest-go"

// Store はデータベースクライアントを保持します
type Store struct {
	client *postgrest.Client
}

// NewStore は新しいStoreインスタンスを作成します
func NewStore(client *postgrest.Client) *Store {
	return &Store{client: client}
}
