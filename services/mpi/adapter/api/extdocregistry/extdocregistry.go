package extdocregistry

import (
	"context"

	fhirModel "wasfaty.api/pkg/fhir/model"
	"wasfaty.api/pkg/log"
	"wasfaty.api/services/mpi/entity"
)

type Client struct {
}

func NewClient() *Client {
	return new(Client)
}

func (c *Client) Search(ctx context.Context, i *fhirModel.Identifier) (*entity.ExtDocRegistrySearchResult, error) {
	var code string
	if len(i.Type.Codings) > 0 {
		code = i.Type.Codings[0].Code
	}

	log.InfoF(ctx, "sending request to external registry for %s:%s", code, i.Value)

	return &entity.ExtDocRegistrySearchResult{IsValid: true}, nil
}
