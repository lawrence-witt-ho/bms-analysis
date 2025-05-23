package handler

import (
	"context"
	"fmt"

	"github.com/atoscerebro/bms-analysis/internal/config"
	"github.com/atoscerebro/bms-analysis/internal/kibana"
)

type Handler struct {
	ctx          context.Context
	config       *config.Config
	kibanaClient *kibana.KibanaClient
}

func NewHandler(cfg *config.Config) *Handler {
	return &Handler{
		config:       cfg,
		kibanaClient: kibana.NewKibanaClient(cfg),
	}
}

func (a *Handler) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *Handler) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// type GetDataResponse struct {
// 	Logs *kibana.KibanaErrorLogs `json:"logs"`
// }

// func (a *Handler) GetData() (*GetDataResponse, error) {
// 	logs, err := a.kibanaClient.GetErrors()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &GetDataResponse{
// 		Logs: logs,
// 	}, nil
// }
