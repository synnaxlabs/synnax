// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PayloadAction } from "@reduxjs/toolkit";
import { type Synnax } from "@synnaxlabs/client";
import { Status, Synnax as PSynnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { migrate } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { useDispatch } from "react-redux";

export interface UseLoadRemoteProps<V extends migrate.Migratable> {
  name: string;
  targetVersion: string;
  layoutKey: string;
  useSelectVersion: (layoutKey: string) => string | undefined;
  fetcher: (client: Synnax, layoutKey: string) => Promise<V>;
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
  const addStatus = Status.useAggregator();
  const client = PSynnax.use();
  const get = useMutation({
    mutationKey: [layoutKey, client?.key],
    mutationFn: async () => {
      if (client == null) return;
      return fetcher(client, layoutKey);
    },
    onError: (e) => Status.handleException(e, `Failed to load ${name}`, addStatus),
  });
  const versionPresent = version != null;
  const notOutdated = versionPresent && !migrate.semVerOlder(version, targetVersion);
  useAsyncEffect(async () => {
    // If the layout data already exists and is not outdated, don't fetch.
    if (notOutdated) return;
    const res = await get.mutateAsync();
    if (res == null) return;
    dispatch(actionCreator(res));
  }, [get.mutate, notOutdated, layoutKey, targetVersion]);
  // If the layout data is null or outdated, return null.
  if (version == null || migrate.semVerOlder(version, targetVersion)) return null;
  return version != null;
};
