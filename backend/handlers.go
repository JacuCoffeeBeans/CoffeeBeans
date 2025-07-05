package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

// コーヒー豆のデータ構造を定義する
type Bean struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Origin string `json:"origin"`
	Price  int    `json:"price"`
}

// ダミーデータのリスト（仮のデータベース）
var beans = []Bean{
	{ID: 1, Name: "エチオピア イルガチェフェ", Origin: "エチオピア", Price: 1200},
	{ID: 2, Name: "ブラジル サントスNo.2", Origin: "ブラジル", Price: 800},
	{ID: 3, Name: "グアテマラ SHB", Origin: "グアテマラ", Price: 1000},
}

// GET /api/beans のリクエストを処理するハンドラ
func getBeansHandler(w http.ResponseWriter, r *http.Request) {
	// レスポンスの形式がJSONであることをヘッダーで指定
	w.Header().Set("Content-Type", "application/json")

	// beansスライスをJSONに変換してレスポンスとして書き出す
	if err := json.NewEncoder(w).Encode(beans); err != nil {
		// JSONへの変換でエラーが起きたら、サーバーエラーを返す
		http.Error(w, "Failed to encode beans to JSON", http.StatusInternalServerError)
	}
}

// findBeanByID は、指定されたIDを持つBeanを検索します。
// 見つかった場合はBeanへのポインタとtrueを、見つからない場合はnilとfalseを返します。
func findBeanByID(id int) (*Bean, bool) {
	for i := range beans {
		if beans[i].ID == id {
			return &beans[i], true
		}
	}
	return nil, false
}

// GET /api/beans/{id} のリクエストを処理するハンドラ
func getBeanHandler(w http.ResponseWriter, r *http.Request) {
	// URLからID部分を取得
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		// IDが数値でなければ、400 Bad Requestエラーを返す
		http.Error(w, "Invalid bean ID", http.StatusBadRequest)
		return
	}

	// ヘルパー関数を使って豆を検索
	bean, found := findBeanByID(id)
	if !found {
		// 見つからなかった場合、404 Not Foundエラーを返す
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(bean); err != nil {
		log.Printf("ERROR: Failed to encode bean (id: %d) to JSON: %v", id, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
