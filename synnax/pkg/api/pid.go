package api

import (
	"context"
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/workspace/pid"
	"github.com/synnaxlabs/x/gorp"
	"go/types"
)

type PIDService struct {
	dbProvider
	internal *pid.Service
}

func NewPIDService(p Provider) *PIDService {
	return &PIDService{
		dbProvider: p.db,
		internal:   p.Config.PID,
	}
}

type PIDCreateRequest struct {
	Workspace uuid.UUID `json:"workspace" msgpack:"workspace"`
	PIDs      []pid.PID `json:"pids" msgpack:"pids"`
}

type PIDCreateResponse struct {
	PIDs []pid.PID `json:"pids" msgpack:"pids"`
}

func (s *PIDService) Create(ctx context.Context, req PIDCreateRequest) (res PIDCreateResponse, err errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		for _, pid_ := range req.PIDs {
			err := s.internal.NewWriter(tx).Create(ctx, req.Workspace, &pid_)
			if err != nil {
				return errors.MaybeQuery(err)
			}
			res.PIDs = append(res.PIDs, pid_)
		}
		return errors.Nil
	})
}

type PIDRenameRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key"`
	Name string    `json:"name" msgpack:"name"`
}

func (s *PIDService) Rename(ctx context.Context, req PIDRenameRequest) (res types.Nil, err errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		err := s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
		return errors.MaybeQuery(err)
	})
}

type PIDSetDataRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key"`
	Data string    `json:"data" msgpack:"data"`
}

func (s *PIDService) SetData(ctx context.Context, req PIDSetDataRequest) (res types.Nil, err errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		err := s.internal.NewWriter(tx).SetData(ctx, req.Key, req.Data)
		return errors.MaybeQuery(err)
	})
}

type PIDRetrieveRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

type PIDRetrieveResponse struct {
	PIDs []pid.PID `json:"pids" msgpack:"pids"`
}

func (s *PIDService) Retrieve(ctx context.Context, req PIDRetrieveRequest) (res PIDRetrieveResponse, err errors.Typed) {
	err = errors.MaybeQuery(s.internal.NewRetrieve().
		WhereKeys(req.Keys...).Entries(&res.PIDs).Exec(ctx, nil))
	return res, err
}

type PIDDeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *PIDService) Delete(ctx context.Context, req PIDDeleteRequest) (res types.Nil, err errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		err := s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
		return errors.MaybeQuery(err)
	})
}
