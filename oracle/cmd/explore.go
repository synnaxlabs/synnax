// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/synnaxlabs/oracle/explore"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/x/errors"
)

const exploreAddrFlag = "addr"

func newExploreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "explore",
		Short: "Serve a browsable map of every schema type",
		Long: `Serve a page that maps every schema type, its fields, and the references
between types.

The page analyzes the schemas again on each reload, so it shows edits without a
restart.`,
		Args: cobra.NoArgs,
		RunE: runExplore,
	}
	cmd.Flags().String(exploreAddrFlag, "localhost:7420", "Address to serve on")
	return cmd
}

func runExplore(cmd *cobra.Command, _ []string) error {
	repoRoot, err := paths.RepoRoot()
	if err != nil {
		return errors.Wrap(err, "explore must be run within a git repository")
	}
	load := func(ctx context.Context) (explore.Model, error) {
		schemas, err := pipeline.DiscoverSchemas(repoRoot)
		if err != nil {
			return explore.Model{}, err
		}
		res, err := pipeline.Run(
			ctx,
			pipeline.Options{RepoRoot: repoRoot, Schemas: schemas},
		)
		if err != nil {
			return explore.Model{}, err
		}
		if res.Resolutions == nil {
			return explore.Model{}, errors.Wrap(
				res.Diagnostics.Flat(),
				"analysis failed",
			)
		}
		return explore.Build(res.Resolutions, res.Chains), nil
	}
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer stop()
	if _, err := load(ctx); err != nil {
		return err
	}
	addr := lo.Must(cmd.Flags().GetString(exploreAddrFlag))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	srv := &http.Server{
		Handler:           explore.Handler(load),
		ReadHeaderTimeout: 5 * time.Second,
	}
	shutdown := make(chan error, 1)
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		shutdown <- srv.Shutdown(shutdownCtx)
	}()
	url := fmt.Sprintf("http://%s", listener.Addr())
	if _, err := fmt.Fprintln(
		cmd.OutOrStdout(),
		"Serving the schema map at",
		url,
	); err != nil {
		return err
	}
	if err := srv.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return <-shutdown
}
