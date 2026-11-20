// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, type panel, type rack } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { Arc } from "@synnaxlabs/pluto";

import { useCreateModal } from "@/platform/arc/useCreateModal";
import { Panel } from "@/platform/panel";

export interface UseCreateProps {
  tabKey?: panel.TabKey;
  /** The rack to bind a created automation to. */
  rack?: rack.Key;
}

export const useCreate = ({ tabKey, rack }: UseCreateProps = {}): (() => void) => {
  const openModal = useCreateModal();
  const openTab = Panel.useOpenTab();
  const handleError = Status.useErrorHandler();
  const { updateAsync: updateTask } = Arc.useUpdateTask();
  return () => {
    handleError(async () => {
      const result = await openModal({});
      if (result == null) return;
      if (rack != null && result.type === "automation")
        await updateTask({ key: result.key, rack });
      openTab({
        variant: "resource",
        resource: arc.resourceID(result.key),
        key: tabKey,
      });
    }, "Failed to create automation");
  };
};
