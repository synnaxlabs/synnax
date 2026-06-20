// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc } from "@synnaxlabs/client";
import { Arc, type Flux } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { useCreateModal } from "@/arc/editor/CreateModal";
import { create } from "@/arc/editor/layout";
import { ZERO_GRAPH } from "@/arc/slice";
import { Layout } from "@/layout";

interface CreateArgs {
  key?: string;
  name?: string;
  mode?: arc.Mode;
}

export const useCreate = (): ((args?: CreateArgs) => void) => {
  const openModal = useCreateModal();
  const placeLayout = Layout.usePlacer();
  const { update } = Arc.useCreate({
    beforeUpdate: useCallback(
      async ({ data }: Flux.BeforeUpdateParams<arc.New>) => {
        const result = await openModal({});
        if (result == null) return false;
        return { ...data, ...result };
      },
      [openModal],
    ),
    afterSuccess: useCallback(
      ({ data }: Flux.AfterSuccessParams<arc.Arc>) => {
        placeLayout(
          create({
            key: data.key,
            name: data.name,
            mode: data.mode,
            remoteCreated: true,
          }),
        );
      },
      [placeLayout],
    ),
  });
  return useCallback(
    (args: CreateArgs = {}) =>
      void update({
        key: args.key,
        name: args.name ?? "Arc Editor",
        mode: args.mode ?? "graph",
        graph: ZERO_GRAPH,
        text: { raw: "" },
      }),
    [update],
  );
};
