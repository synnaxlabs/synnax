// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PayloadAction } from "@reduxjs/toolkit";
import { DisconnectedError, type Synnax as Client } from "@synnaxlabs/client";
import { Status, Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { migrate } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { useDispatch } from "react-redux";

export interface UseLoadRemoteProps<V extends migrate.Migratable> {
  name: string;
  targetVersion: string;
  layoutKey: string;
  useSelectVersion: (layoutKey: string) => string | undefined;
  fetcher: (client: Client, layoutKey: string) => Promise<V>;
  actionCreator: (v: V) => PayloadAction<any>;
}

export const useLoadRemote = <V extends migrate.Migratable>({
  name,
  targetVersion,
  layoutKey,
  useSelectVersion,
  fetcher,
  actionCreator,
}: UseLoadRemoteProps<V>): boolean | null => {
  const dispatch = useDispatch();
  const version = useSelectVersion(layoutKey);
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();
  const get = useMutation({
    mutationFn: async () => {
      if (client == null) throw new DisconnectedError();
      return fetcher(client, layoutKey);
    },
    onError: (e) => handleError(e, `Failed to load ${name}`),
  });
  const versionPresent = version != null;
  const notOutdated = versionPresent && !migrate.semVerOlder(version, targetVersion);
  useAsyncEffect(
    async (signal) => {
      // If the layout data already exists and is not outdated, don't fetch.
      if (notOutdated) return;
      const res = await get.mutateAsync();
      if (signal.aborted) return;
      if (res == null) return;
      dispatch(actionCreator(res));
    },
    [get.mutate, notOutdated, layoutKey, targetVersion],
  );
  // If the layout data is null or outdated, return null.
  if (version == null || migrate.semVerOlder(version, targetVersion)) return null;
  return version != null;
};
