// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package session

import (
	"context"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/cluster"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the session API service.
type ServiceConfig struct {
	// Auth logs subjects in and stores their sessions.
	//
	// [REQUIRED]
	Auth *svcauth.Service
	// User stores and retrieves users.
	//
	// [REQUIRED]
	User *user.Service
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
	// Cluster provides information about the cluster topology.
	//
	// [REQUIRED]
	Cluster cluster.Cluster
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the session API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Auth = override.Nil(c.Auth, other.Auth)
	c.User = override.Nil(c.User, other.User)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.Cluster = override.Nil(c.Cluster, other.Cluster)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.auth.session")
	v.NotNil("auth", c.Auth)
	v.NotNil("user", c.User)
	v.NotNil("rbac", c.RBAC)
	v.NotNil("cluster", c.Cluster)
	return v.Error()
}

// Service logs subjects in and lists and ends their sessions.
type Service struct {
	auth    *svcauth.Service
	user    *user.Service
	rbac    *rbac.Service
	cluster cluster.Cluster
}

// NewService opens a new session API service.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		auth:    cfg.Auth,
		user:    cfg.User,
		rbac:    cfg.RBAC,
		cluster: cfg.Cluster,
	}, nil
}

// LoginResponse is the session a password opened.
type LoginResponse struct {
	// User is the user the token is associated with.
	User user.User `json:"user" msgpack:"user"`
	// Token is the session token to send with every later request.
	Token string `json:"token" msgpack:"token"`
	// ClusterInfo is the information about the cluster.
	ClusterInfo auth.ClusterInfo `json:"cluster_info" msgpack:"cluster_info"`
}

// LoginRequest logs a user in with a username and password.
type LoginRequest struct{ svcauth.Credentials }

// Login opens a session for the user with the provided credentials. If successful,
// returns a response containing the session token along with the user's details. It
// is kept for clients built before Create existed.
func (s *Service) Login(
	ctx context.Context,
	req LoginRequest,
) (LoginResponse, error) {
	start := telem.Now()
	res, err := s.Create(ctx, CreateRequest{Proof: svcauth.Proof{
		Variant: svcauth.PasswordProof{Credentials: req.Credentials},
	}})
	if err != nil {
		return LoginResponse{}, err
	}
	return LoginResponse{
		User:        *res.User,
		Token:       res.Token,
		ClusterInfo: s.clusterInfo(start),
	}, nil
}

// clusterInfo stamps the node time at the midpoint since start.
func (s *Service) clusterInfo(start telem.TimeStamp) auth.ClusterInfo {
	end := telem.Now()
	return auth.ClusterInfo{
		ClusterKey:  s.cluster.Key().String(),
		NodeKey:     s.cluster.HostKey(),
		NodeVersion: version.Get(),
		NodeTime:    start + (end-start)/2,
	}
}

// CreateRequest logs a subject in with a proof.
type CreateRequest struct {
	// Proof is what the subject presents. Its method selects the authenticator.
	Proof svcauth.Proof `json:"proof" msgpack:"proof"`
	// Cookie asks for the token in a cookie that page scripts cannot read, in place of
	// the response body. A browser client sets it. Only the HTTP transport honors it.
	Cookie bool `json:"cookie" msgpack:"cookie"`
}

// CreateResponse is the session a proof opened.
type CreateResponse struct {
	// Token is the session token to send with every later request.
	Token string `json:"token" msgpack:"token"`
	// Session is the session the proof opened.
	Session svcauth.Session `json:"session" msgpack:"session"`
	// User is the user record of the subject, when the subject is a user.
	User *user.User `json:"user,omitempty" msgpack:"user,omitempty"`
}

// Create logs the subject in with the proof in req and returns the session, its token,
// and the user record when the subject is a user. With req.Cookie set, the token goes
// to the transport through [auth.SessionCookieParam] and the body carries none.
func (s *Service) Create(
	ctx context.Context,
	req CreateRequest,
) (CreateResponse, error) {
	session, tk, err := s.auth.Login(ctx, req.Proof)
	if err != nil {
		return CreateResponse{}, err
	}
	res := CreateResponse{
		Token:   auth.Deliver(ctx, s.auth, tk, req.Cookie),
		Session: session,
	}
	if res.User, err = s.userOf(ctx, session.Subject); err != nil {
		return CreateResponse{}, err
	}
	return res, nil
}

// RenewResponse is the caller's session, so a client that holds only a token learns who
// it is.
type RenewResponse struct {
	// Session is the caller's session.
	Session svcauth.Session `json:"session" msgpack:"session"`
	// User is the user record of the subject, when the subject is a user.
	User *user.User `json:"user,omitempty" msgpack:"user,omitempty"`
}

// Renew reports activity on the caller's session and returns it. The middleware
// records the activity when it checks the token.
func (s *Service) Renew(
	ctx context.Context,
	_ struct{},
) (RenewResponse, error) {
	session := auth.GetSession(ctx)
	u, err := s.userOf(ctx, session.Subject)
	if err != nil {
		return RenewResponse{}, err
	}
	return RenewResponse{Session: session, User: u}, nil
}

func (s *Service) userOf(ctx context.Context, subject ontology.ID) (*user.User, error) {
	if subject.Type != ontology.ResourceTypeUser {
		return nil, nil
	}
	key, err := uuid.Parse(subject.Key)
	if err != nil {
		return nil, err
	}
	var u user.User
	if err := s.user.NewRetrieve().
		Where(user.MatchKeys(key)).
		Entry(&u).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	return &u, nil
}

// RetrieveRequest selects sessions to list.
type RetrieveRequest struct {
	// Subjects limits the result to these subjects' sessions. Empty means the caller.
	Subjects []ontology.ID `json:"subjects" msgpack:"subjects"`
}

// RetrieveResponse lists sessions.
type RetrieveResponse struct {
	// Sessions are the sessions that matched.
	Sessions []svcauth.Session `json:"sessions" msgpack:"sessions"`
}

// Retrieve lists the caller's sessions, or the sessions of the given subjects when the
// caller may retrieve those subjects.
func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	caller := auth.GetSubject(ctx)
	subjects := req.Subjects
	if len(subjects) == 0 {
		subjects = []ontology.ID{caller}
	}
	if err := auth.EnforceOthers(
		ctx,
		s.rbac,
		caller,
		access.ActionRetrieve,
		subjects,
	); err != nil {
		return RetrieveResponse{}, err
	}
	sessions := make([]svcauth.Session, 0)
	if err := s.auth.NewRetrieve().
		Where(svcauth.MatchSubjects(subjects...)).
		Entries(&sessions).
		Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Sessions: sessions}, nil
}

// DeleteRequest names sessions to end.
type DeleteRequest struct {
	// Keys are the sessions to end.
	Keys []svcauth.SessionKey `json:"keys" msgpack:"keys"`
}

// Delete ends the given sessions. A caller may end its own sessions; ending another
// subject's needs permission to delete that subject. Unknown keys are ignored. Ending
// the caller's own session asks the transport to clear its cookie, if it set one.
func (s *Service) Delete(
	ctx context.Context,
	req DeleteRequest,
) (struct{}, error) {
	var sessions []svcauth.Session
	if err := s.auth.NewRetrieve().
		Where(svcauth.MatchKeys(req.Keys...)).
		Entries(&sessions).
		Exec(ctx, nil); err != nil && !errors.Is(err, query.ErrNotFound) {
		return struct{}{}, err
	}
	subjects := lo.Uniq(lo.Map(sessions, func(e svcauth.Session, _ int) ontology.ID {
		return e.Subject
	}))
	if err := auth.EnforceOthers(
		ctx,
		s.rbac,
		auth.GetSubject(ctx),
		access.ActionDelete,
		subjects,
	); err != nil {
		return struct{}{}, err
	}
	if lo.Contains(req.Keys, auth.GetSession(ctx).Key) {
		freighter.MDFromContext(ctx).Set(auth.SessionCookieParam, auth.SessionCookie{})
	}
	return struct{}{}, s.auth.Logout(ctx, req.Keys...)
}
