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

import { create } from "@/layered/service/arc/layout";
import { useCreateModal } from "@/layered/service/arc/useCreateModal";
import { Layout } from "@/layout";

interface CreateArgs extends Partial<Pick<arc.New, "key" | "name" | "mode">> {}

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
      ({ data: { key, name } }: Flux.AfterSuccessParams<arc.Arc>) => {
        placeLayout(create({ key, name }));
      },
      [placeLayout],
    ),
  });
  return useCallback(
    (args: CreateArgs = {}) => update({ name: "Arc Editor", mode: "graph", ...args }),
    [update],
  );
};
