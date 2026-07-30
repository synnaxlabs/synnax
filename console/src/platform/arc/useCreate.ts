// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, type panel } from "@synnaxlabs/client";
import { Arc, type Flux } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { useCreateModal } from "@/platform/arc/useCreateModal";
import { Panel } from "@/platform/panel";

export interface UseCreateProps {
  tabKey?: panel.TabKey;
}

export const useCreate = ({ tabKey }: UseCreateProps = {}): (() => void) => {
  const openModal = useCreateModal();
  const openTab = Panel.useOpenTab();
  const { update } = Arc.useCreate({
    beforeUpdate: useCallback(
      async ({ data }: Flux.BeforeUpdateParams<arc.New>) => {
        const result = await openModal({});
        if (result == null) return false;
        return { ...data, ...result };
      },
      [openModal],
    ),
    afterOptimistic: useCallback(
      ({ data: { key } }: Flux.AfterSuccessParams<arc.Arc>) =>
        openTab({ variant: "resource", resource: arc.ontologyID(key), key: tabKey }),
      [openTab, tabKey],
    ),
  });
  return useCallback(() => update({ name: "Arc Editor", mode: "graph" }), [update]);
};
