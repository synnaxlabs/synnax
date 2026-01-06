// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PayloadAction } from "@reduxjs/toolkit";
import { type Synnax as Client } from "@synnaxlabs/client";
import { type Flux, type state, useAsyncEffect } from "@synnaxlabs/pluto";
import { migrate } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

export interface UseLoadRemoteProps<V extends migrate.Migratable> {
  name: string;
  targetVersion: string;
  layoutKey: string;
  useSelectVersion: (layoutKey: string) => string | undefined;
  fetcher: (client: Client, layoutKey: string) => Promise<V>;
  actionCreator: (v: V) => PayloadAction<any>;
}

interface RetrieveParams {
  key: string;
}

interface CreateLoadRemoteParams<V extends state.State> {
  targetVersion: string;
  useRetrieve: Flux.UseRetrieveObservable<RetrieveParams, V>;
  useSelectVersion: (layoutKey: string) => string | undefined;
  actionCreator: (v: V) => PayloadAction<any>;
}

export const createLoadRemote =
  <V extends state.State>({
    targetVersion,
    useSelectVersion,
    useRetrieve,
    actionCreator,
  }: CreateLoadRemoteParams<V>) =>
  (layoutKey: string) => {
    const dispatch = useDispatch();
    const version = useSelectVersion(layoutKey);
    const { retrieve } = useRetrieve({
      onChange: useCallback(
        (result) => {
          if (result.variant !== "success") return;
          dispatch(actionCreator(result.data));
        },
        [dispatch, actionCreator],
      ),
    });
    const versionPresent = version != null;
    const notOutdated = versionPresent && !migrate.semVerOlder(version, targetVersion);
    useAsyncEffect(
      async (signal) => {
        // If the layout data already exists and is not outdated, don't fetch.
        if (notOutdated) return;
        retrieve({ key: layoutKey }, { signal });
      },
      [retrieve, notOutdated, layoutKey, targetVersion],
    );
    // If the layout data is null or outdated, return null.
    if (version == null || migrate.semVerOlder(version, targetVersion)) return null;
    return version != null;
  };
