package monarch

import (
	"context"

	"github.com/localitas/localitas-app-mmoney/internal/monarchgql"
	"github.com/localitas/localitas-app-mmoney/internal/queries"
)

var GetCategoriesQuery = queries.Get("categories/list.graphql")

type Category struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	GroupName string `json:"group_name"`
	GroupID   string `json:"group_id"`
	GroupType string `json:"group_type"`
	Order     int    `json:"order"`
	Icon      string `json:"icon"`
}

func (s *Service) ListCategories(ctx context.Context) ([]Category, error) {
	var resp struct {
		Categories []struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Order int    `json:"order"`
			Icon  string `json:"icon"`
			Group struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"group"`
		} `json:"categories"`
	}

	err := s.Client.Do(ctx, &monarchgql.Request{
		OperationName: "GetCategories",
		Query:         GetCategoriesQuery,
	}, &resp)
	if err != nil {
		return nil, err
	}

	cats := make([]Category, len(resp.Categories))
	for i, c := range resp.Categories {
		cats[i] = Category{
			ID:        c.ID,
			Name:      c.Name,
			GroupName: c.Group.Name,
			GroupID:   c.Group.ID,
			GroupType: c.Group.Type,
			Order:     c.Order,
			Icon:      c.Icon,
		}
	}
	return cats, nil
}
