// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth/token"
	svcbackup "github.com/synnaxlabs/synnax/pkg/service/backup"
	"github.com/synnaxlabs/synnax/pkg/service/user"
)

// Transport implements fhttp.BindableTransport to serve .sy backup files
// via a raw Fiber handler, bypassing Freighter's JSON serialization.
type Transport struct {
	internal *svcbackup.Service
	token    *token.Service
	access   *rbac.Service
	sessions *sessionStore
}

var _ fhttp.BindableTransport = (*Transport)(nil)

// NewTransport creates a new backup transport. The caller must call Close
// when shutting down to stop the session cleanup goroutine.
func NewTransport(
	svc *svcbackup.Service,
	tokenSvc *token.Service,
	accessSvc *rbac.Service,
) *Transport {
	ctx, cancel := context.WithCancel(context.Background())
	t := &Transport{
		internal: svc,
		token:    tokenSvc,
		access:   accessSvc,
		sessions: newSessionStore(cancel),
	}
	go t.sessions.cleanupLoop(ctx)
	return t
}

// Close stops the session cleanup goroutine.
func (t *Transport) Close() { t.sessions.cancel() }

// BindTo registers the backup endpoints on the Fiber app.
func (t *Transport) BindTo(app *fiber.App) {
	app.Post("/api/v1/export", t.handleExport)
	app.Post("/api/v1/import/analyze", t.handleAnalyze)
	app.Post("/api/v1/import", t.handleImport)
}

func (t *Transport) authenticate(c fiber.Ctx) (uuid.UUID, error) {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return uuid.Nil, c.Status(fiber.StatusUnauthorized).
			JSON(fiber.Map{"error": "missing authorization header"})
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	userKey, err := t.token.Validate(tokenStr)
	if err != nil {
		return uuid.Nil, c.Status(fiber.StatusUnauthorized).
			JSON(fiber.Map{"error": "invalid token"})
	}
	return userKey, nil
}

func (t *Transport) handleExport(c fiber.Ctx) error {
	userKey, err := t.authenticate(c)
	if err != nil {
		return err
	}

	var httpReq struct {
		svcbackup.ExportRequest
		Path string `json:"path"`
	}
	if err := json.Unmarshal(c.Body(), &httpReq); err != nil {
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := t.access.Enforce(c.Context(), access.Request{
		Subject: user.OntologyID(userKey),
		Action:  access.ActionRetrieve,
		Objects: httpReq.ExportRequest.OntologyIDs(),
	}); err != nil {
		return c.Status(fiber.StatusForbidden).
			JSON(fiber.Map{"error": err.Error()})
	}

	if httpReq.Path != "" {
		f, err := os.Create(httpReq.Path)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).
				JSON(fiber.Map{"error": "failed to create file: " + err.Error()})
		}
		defer f.Close()
		if err := t.internal.Export(c.Context(), httpReq.ExportRequest, f); err != nil {
			os.Remove(httpReq.Path)
			return c.Status(fiber.StatusInternalServerError).
				JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"path": httpReq.Path})
	}

	c.Set("Content-Type", "application/zip")
	c.Set("Content-Disposition", "attachment; filename=\"backup.sy\"")
	return t.internal.Export(c.Context(), httpReq.ExportRequest, c.Response().BodyWriter())
}

func (t *Transport) handleAnalyze(c fiber.Ctx) error {
	if _, err := t.authenticate(c); err != nil {
		return err
	}

	body := c.Body()
	if len(body) == 0 {
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"error": "empty request body"})
	}

	reader := bytes.NewReader(body)
	resp, err := t.internal.Analyze(c.Context(), reader, int64(len(body)))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": err.Error()})
	}

	sessionID := t.sessions.save(body)
	resp.SessionID = sessionID
	return c.JSON(resp)
}

func (t *Transport) handleImport(c fiber.Ctx) error {
	userKey, err := t.authenticate(c)
	if err != nil {
		return err
	}

	var req svcbackup.ImportRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"error": "invalid request body"})
	}

	data, ok := t.sessions.get(req.SessionID)
	if !ok {
		return c.Status(fiber.StatusNotFound).
			JSON(fiber.Map{"error": "session not found or expired"})
	}

	// Enforce RBAC — import creates and modifies resources, so require
	// ActionCreate. We analyze the archive to determine which ontology
	// resources will be affected.
	reader := bytes.NewReader(data)
	analyzeResp, err := t.internal.Analyze(c.Context(), reader, int64(len(data)))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": err.Error()})
	}
	ontologyIDs := svcbackup.OntologyIDsFromAnalysis(analyzeResp)
	if len(ontologyIDs) > 0 {
		if err := t.access.Enforce(c.Context(), access.Request{
			Subject: user.OntologyID(userKey),
			Action:  access.ActionCreate,
			Objects: ontologyIDs,
		}); err != nil {
			return c.Status(fiber.StatusForbidden).
				JSON(fiber.Map{"error": err.Error()})
		}
	}

	reader = bytes.NewReader(data)
	resp, err := t.internal.Import(c.Context(), reader, int64(len(data)), req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": err.Error()})
	}

	t.sessions.remove(req.SessionID)
	return c.JSON(resp)
}

// Use implements freighter.Transport.
func (*Transport) Use(...freighter.Middleware) {}

// Report implements alamos.ReportProvider.
func (*Transport) Report() alamos.Report {
	return alamos.Report{"backup": "enabled"}
}

// sessionStore holds uploaded .sy archives in memory between the analyze and
// import phases. Sessions expire after 30 minutes.
type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]sessionEntry
	cancel   context.CancelFunc
}

type sessionEntry struct {
	data      []byte
	createdAt time.Time
}

const sessionTTL = 30 * time.Minute

func newSessionStore(cancel context.CancelFunc) *sessionStore {
	return &sessionStore{
		sessions: make(map[string]sessionEntry),
		cancel:   cancel,
	}
}

func (s *sessionStore) save(data []byte) string {
	id := uuid.New().String()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[id] = sessionEntry{data: data, createdAt: time.Now()}
	return id
}

func (s *sessionStore) get(id string) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.sessions[id]
	if !ok {
		return nil, false
	}
	if time.Since(entry.createdAt) > sessionTTL {
		delete(s.sessions, id)
		return nil, false
	}
	return entry.data, true
}

func (s *sessionStore) remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

func (s *sessionStore) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			for id, entry := range s.sessions {
				if time.Since(entry.createdAt) > sessionTTL {
					delete(s.sessions, id)
				}
			}
			s.mu.Unlock()
		}
	}
}
