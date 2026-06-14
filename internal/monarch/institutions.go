package monarch

import (
	"context"

	"github.com/localitas/localitas-app-mmoney/internal/monarchgql"
	"github.com/localitas/localitas-app-mmoney/internal/queries"
)

var GetInstitutionsQuery = queries.Get("institutions/list.graphql")

type Institution struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (s *Service) ListInstitutions(ctx context.Context) ([]Institution, error) {
	var resp struct {
		Credentials []struct {
			Institution struct {
				ID                 string `json:"id"`
				PlaidInstitutionID string `json:"plaidInstitutionId"`
				Name               string `json:"name"`
			} `json:"institution"`
		} `json:"credentials"`
	}

	err := s.Client.Do(ctx, &monarchgql.Request{
		OperationName: "GetInstitutionSettings",
		Query:         GetInstitutionsQuery,
	}, &resp)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	insts := make([]Institution, 0)
	for _, cred := range resp.Credentials {
		inst := cred.Institution
		if inst.Name != "" && !seen[inst.ID] {
			seen[inst.ID] = true
			insts = append(insts, Institution{
				ID:   inst.ID,
				Name: inst.Name,
				URL:  inst.PlaidInstitutionID,
			})
		}
	}
	return insts, nil
}
