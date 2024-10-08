// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PayloadAction } from "@reduxjs/toolkit";
import { Synnax } from "@synnaxlabs/client";
import { Status, Synnax as PSynnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { migrate } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { useDispatch } from "react-redux";

export interface UseLoadRemoteProps<V extends migrate.Migratable> {
  name: string;
  targetVersion: string;
  layoutKey: string;
  useSelect: (layoutKey: string) => V | undefined;
  fetcher: (client: Synnax, layoutKey: string) => Promise<V>;
  actionCreator: (v: V) => PayloadAction<any>;
}

export const useLoadRemote = <V extends migrate.Migratable>({
  name,
  targetVersion,
  layoutKey,
  useSelect,
  fetcher,
  actionCreator,
}: UseLoadRemoteProps<any>): V | null => {
  const dispatch = useDispatch();
  const v = useSelect(layoutKey);
  const addStatus = Status.useAggregator();
  const client = PSynnax.use();
  const get = useMutation({
    mutationKey: [layoutKey, client?.key],
    mutationFn: async () => {
      if (client == null) return;
      return fetcher(client, layoutKey);
    },
    onError: (e) =>
      addStatus({
        variant: "error",
        message: `Failed to load ${name}`,
        description: e.message,
      }),
  });
  useAsyncEffect(async () => {
    // If the layout data already exists and is not outdated, don't fetch.
    if (v != null && !migrate.semVerOlder(v.version, targetVersion)) return;
    dispatch(actionCreator(await get.mutateAsync()));
  }, [get.mutate, v, layoutKey, targetVersion]);
  // If the layout data is null or outdated, return null.
  if (v == null || migrate.semVerOlder(v.version, targetVersion)) return null;
  return v;
};
