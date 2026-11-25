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
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/cluster"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
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
	// DB opens the snapshots that user lookups read.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Auth logs subjects in and stores their sessions.
	//
	// [REQUIRED]
	Auth *auth.Service
	// Password gives the username that session responses send for a user.
	//
	// [REQUIRED]
	Password *password.Service
	// User stores and retrieves users.
	//
	// [REQUIRED]
	User *user.Service
	// Identity names the Core in login responses.
	//
	// [REQUIRED]
	Identity cluster.Identity
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Auth = override.Nil(c.Auth, other.Auth)
	c.Password = override.Nil(c.Password, other.Password)
	c.User = override.Nil(c.User, other.User)
	c.Identity = override.If(
		c.Identity,
		other.Identity,
		other.Identity != cluster.Identity{},
	)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.auth.session")
	v.NotNil("db", c.DB)
	v.NotNil("auth", c.Auth)
	v.NotNil("password", c.Password)
	v.NotNil("user", c.User)
	v.Ternary("identity.key", c.Identity.Key == uuid.Nil(), "must be set")
	v.NonZero("identity.host_key", c.Identity.HostKey)
	return v.Error()
}

// Service logs subjects in and lists, renews, and ends their sessions.
type Service struct {
	db       *gorp.DB
	auth     *auth.Service
	password *password.Service
	user     *user.Service
	identity cluster.Identity
}

// NewService opens a new session API service.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:       cfg.DB,
		auth:     cfg.Auth,
		password: cfg.Password,
		user:     cfg.User,
		identity: cfg.Identity,
	}, nil
}

// User is the user record that session responses send. Username, FirstName, and
// LastName let clients from before the user record dropped them parse the response.
type User struct {
	user.User
	// Username is the login name of the user's password, or empty when the user has
	// no password.
	Username string `json:"username" msgpack:"username"`
	// FirstName is the user's name.
	FirstName string `json:"first_name" msgpack:"first_name"`
	// LastName is always empty.
	LastName string `json:"last_name" msgpack:"last_name"`
}

// LoginResponse is the session a password opened.
type LoginResponse struct {
	// User is the user the token is associated with.
	User User `json:"user" msgpack:"user"`
	// Token is the session token to send with every later request.
	Token string `json:"token" msgpack:"token"`
	// ClusterInfo is the information about the cluster.
	ClusterInfo apiauth.ClusterInfo `json:"cluster_info" msgpack:"cluster_info"`
}

// LoginRequest logs a user in with a username and password.
type LoginRequest struct{ auth.Credentials }

// Login opens a session for the user with the provided credentials. If successful,
// returns a response containing the session token along with the user's details. It
// is kept for clients built before Create existed.
func (s *Service) Login(
	ctx context.Context,
	req LoginRequest,
) (LoginResponse, error) {
	start := telem.Now()
	res, err := s.Create(ctx, CreateRequest{Proof: auth.Proof{
		Variant: auth.PasswordProof{Credentials: req.Credentials},
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
func (s *Service) clusterInfo(start telem.TimeStamp) apiauth.ClusterInfo {
	end := telem.Now()
	return apiauth.ClusterInfo{
		ClusterKey:  s.identity.Key.String(),
		NodeKey:     s.identity.HostKey,
		NodeVersion: version.Get(),
		NodeTime:    start + (end-start)/2,
	}
}

// CreateRequest logs a subject in with a proof.
type CreateRequest struct {
	// Proof is what the subject presents. Its method selects the authenticator.
	Proof auth.Proof `json:"proof" msgpack:"proof"`
	// Cookie asks for the token in a cookie that page scripts cannot read, in place of
	// the response body. A browser client sets it. Only the HTTP transport honors it.
	Cookie bool `json:"cookie" msgpack:"cookie"`
}

// CreateResponse is the session a proof opened.
type CreateResponse struct {
	// Token is the session token to send with every later request.
	Token string `json:"token" msgpack:"token"`
	// Session is the session the proof opened.
	Session auth.Session `json:"session" msgpack:"session"`
	// User is the user record of the subject, when the subject is a user.
	User *User `json:"user,omitempty" msgpack:"user,omitempty"`
}

// Create logs the subject in with the proof in req and returns the session, its token,
// and the user record when the subject is a user. With req.Cookie set, the token goes
// to the transport through [apiauth.SessionCookieParam] and the body carries none.
func (s *Service) Create(
	ctx context.Context,
	req CreateRequest,
) (CreateResponse, error) {
	session, tk, err := s.auth.Login(ctx, req.Proof)
	if err != nil {
		return CreateResponse{}, err
	}
	res := CreateResponse{
		Token:   apiauth.Deliver(ctx, s.auth, tk, req.Cookie),
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
	Session auth.Session `json:"session" msgpack:"session"`
	// User is the user record of the subject, when the subject is a user.
	User *User `json:"user,omitempty" msgpack:"user,omitempty"`
}

// Renew reports activity on the caller's session and returns it. The middleware
// records the activity when it checks the token.
func (s *Service) Renew(
	ctx context.Context,
	_ struct{},
) (RenewResponse, error) {
	session := apiauth.GetSession(ctx)
	u, err := s.userOf(ctx, session.Subject)
	if err != nil {
		return RenewResponse{}, err
	}
	return RenewResponse{Session: session, User: u}, nil
}

func (s *Service) userOf(ctx context.Context, subject resource.ID) (*User, error) {
	if subject.Type != resource.TypeUser {
		return nil, nil
	}
	key, err := uuid.Parse(subject.Key)
	if err != nil {
		return nil, err
	}
	var (
		u User
		p password.Password
	)
	if err := s.db.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		if err := s.user.NewRetrieve().
			Where(user.MatchKeys(key)).
			Entry(&u.User).
			Exec(ctx, snap); err != nil {
			return err
		}
		return errors.Skip(s.password.NewRetrieve().
			Where(password.MatchSubjects(subject)).
			Entry(&p).
			Exec(ctx, snap), query.ErrNotFound)
	}); err != nil {
		return nil, err
	}
	u.Username = p.Username
	u.FirstName = u.Name
	return &u, nil
}

// RetrieveRequest selects sessions to list.
type RetrieveRequest struct {
	// Subjects limits the result to these subjects' sessions. Empty means the caller.
	Subjects []resource.ID `json:"subjects" msgpack:"subjects"`
}

// RetrieveResponse lists sessions.
type RetrieveResponse struct {
	// Sessions are the sessions that matched.
	Sessions []auth.Session `json:"sessions" msgpack:"sessions"`
}

// Holders implements bind.Held.
func (r RetrieveRequest) Holders() []resource.ID { return r.Subjects }

// Retrieve lists the sessions of the given subjects in snap, or the caller's sessions
// when the request names no subject.
func (s *Service) Retrieve(
	ctx context.Context,
	snap gorp.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	subjects := req.Subjects
	if len(subjects) == 0 {
		subjects = []resource.ID{apiauth.GetSubject(ctx)}
	}
	sessions := make([]auth.Session, 0)
	if err := s.auth.NewRetrieve().
		Where(auth.MatchSubjects(subjects...)).
		Entries(&sessions).
		Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Sessions: sessions}, nil
}

// DeleteRequest names sessions to end.
type DeleteRequest struct {
	// Keys are the sessions to end.
	Keys []auth.SessionKey `json:"keys" msgpack:"keys"`
}

// Delete ends the given sessions. Unknown keys are ignored. Ending the caller's own
// session asks the transport to clear its cookie, if it set one.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if lo.Contains(req.Keys, apiauth.GetSession(ctx).Key) {
		freighter.MDFromContext(ctx).Set(
			apiauth.SessionCookieParam,
			apiauth.SessionCookie{},
		)
	}
	return struct{}{}, s.auth.NewWriter(tx).Delete(ctx, req.Keys...)
}
