package api

import (
	"context"
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/workspace/lineplot"
	"github.com/synnaxlabs/x/gorp"
	"go/types"
)

type LinePlotService struct {
	dbProvider
	internal *lineplot.Service
}

func NewLinePlotService(p Provider) *LinePlotService {
	return &LinePlotService{
		dbProvider: p.db,
		internal:   p.Config.LinePlot,
	}
}

type LinePlotCreateRequest struct {
	Workspace uuid.UUID           `json:"workspace" msgpack:"workspace"`
	LinePlots []lineplot.LinePlot `json:"line_plots" msgpack:"line_plots"`
}

type LinePlotCreateResponse struct {
	LinePlots []lineplot.LinePlot `json:"line_plots" msgpack:"line_plots"`
}

func (s *LinePlotService) Create(ctx context.Context, req LinePlotCreateRequest) (res LinePlotCreateResponse, err errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		for _, lp := range req.LinePlots {
			err := s.internal.NewWriter(tx).Create(ctx, req.Workspace, &lp)
			if err != nil {
				return errors.MaybeQuery(err)
			}
			res.LinePlots = append(res.LinePlots, lp)
		}
		return errors.Nil
	})
}

type LinePlotRenameRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key"`
	Name string    `json:"name" msgpack:"name"`
}

func (s *LinePlotService) Rename(ctx context.Context, req LinePlotRenameRequest) (res types.Nil, err errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		err := s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
		return errors.MaybeQuery(err)
	})
}

type LinePlotSetDataRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key"`
	Data string    `json:"data" msgpack:"data"`
}

func (s *LinePlotService) SetData(ctx context.Context, req LinePlotSetDataRequest) (res types.Nil, err errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		err := s.internal.NewWriter(tx).SetData(ctx, req.Key, req.Data)
		return errors.MaybeQuery(err)
	})
}

type LinePlotRetrieveRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

type LinePlotRetrieveResponse struct {
	LinePlots []lineplot.LinePlot `json:"line_plots" msgpack:"line_plots"`
}

func (s *LinePlotService) Retrieve(ctx context.Context, req LinePlotRetrieveRequest) (res LinePlotRetrieveResponse, err errors.Typed) {
	err = errors.MaybeQuery(s.internal.NewRetrieve().
		WhereKeys(req.Keys...).Entries(&res.LinePlots).Exec(ctx, nil))
	return res, err
}

type LinePlotDeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *LinePlotService) Delete(ctx context.Context, req LinePlotDeleteRequest) (res types.Nil, err errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		err := s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
		return errors.MaybeQuery(err)
	})
}
