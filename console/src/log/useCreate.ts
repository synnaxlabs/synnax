// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type log, type workspace } from "@synnaxlabs/client";
import { Log as PLog } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Layout } from "@/layout";
import { create } from "@/log/layout";
import { Workspace } from "@/workspace";

export interface UseCreateProps {
  workspace?: workspace.Key;
}

// useCreate returns a callback that creates a log on the server through the Pluto
// flux store and places its layout once the record exists, so the connected
// component can retrieve it without a pendingUpload round-trip.
export const useCreate = ({
  workspace,
}: UseCreateProps): ((init?: Partial<log.New>) => void) => {
  const activeWorkspace = Workspace.useSelectActiveKey();
  const maybeChangeWorkspace = Workspace.useMaybeChange();
  const placeLayout = Layout.usePlacer();
  const { update } = PLog.useCreate({
    afterSuccess: async ({ data }) => {
      const { workspace, key, name } = data;
      if (workspace != null) await maybeChangeWorkspace(workspace);
      placeLayout(create({ key, name }));
    },
  });
  return useCallback(
    (init) =>
      update({
        name: "Log",
        ...init,
        workspace: workspace ?? activeWorkspace ?? undefined,
      }),
    [workspace, activeWorkspace, update],
  );
};
