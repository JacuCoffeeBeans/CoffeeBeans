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
