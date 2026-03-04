package handler

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/Loszect1/Ecommerce---BE-Golang/app"
	"github.com/Loszect1/Ecommerce---BE-Golang/config"
)

var (
	once    sync.Once
	handler http.Handler
)

func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(func() {
		cfg := config.FromEnv()
		handler = app.New(cfg)
	})

	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()

	r = r.WithContext(ctx)
	handler.ServeHTTP(w, r)
}
