package handler

import (
	"context"
	"fmt"
)

type Handler struct {
	ctx context.Context
}

func NewHandler() *Handler {
	return &Handler{}
}

func (a *Handler) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *Handler) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}
