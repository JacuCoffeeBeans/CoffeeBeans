// backend/middleware.go
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey は、コンテキストのキーとして使われる文字列の型です。
// stringの衝突を避けるために独自の型を定義します。
type contextKey string

const userIDKey contextKey = "userID"

// jwtAuthMiddleware は、JWTを検証するミドルウェアです
func jwtAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// リクエストヘッダーから "Authorization" を取得
		authHeader := r.Header.Get("Authorization")

		// ヘッダーが空、またはBearer形式でなければ、
		// 何もせず、そのまま次のハンドラを呼び出す
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		// ヘッダーが "Bearer <token>" の形式になっているか検証
		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}
		tokenString := headerParts[1]

		// トークンをパースして検証
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// 署名メソッドがHMACであることを確認
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			// 環境変数からJWTのシークレットキーを取得
			secret := os.Getenv("SUPABASE_JWT_SECRET")
			return []byte(secret), nil
		})

		// トークンが無効な場合はエラー
		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// トークンからクレーム（特に "sub" クレーム）を取得し、コンテキストにユーザーIDをセット
		claims, ok := token.Claims.(jwt.MapClaims)
		if ok && claims["sub"] != nil {
			ctx := context.WithValue(r.Context(), userIDKey, claims["sub"])
			r = r.WithContext(ctx)
		}

		// 次のハンドラへ処理を渡す
		next.ServeHTTP(w, r)
	})
}
