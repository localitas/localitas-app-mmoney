package monarch

import (
	"context"

	"github.com/localitas/localitas-app-mmoney/internal/monarchgql"
)

type graphQLClient interface {
	Do(ctx context.Context, reqBody *monarchgql.Request, result interface{}) error
	TokenValue() string
}

type Service struct {
	Client graphQLClient
}

func NewService(client graphQLClient) *Service {
	return &Service{Client: client}
}
