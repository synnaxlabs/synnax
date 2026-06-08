// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot, type workspace } from "@synnaxlabs/client";
import { LinePlot } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Layout } from "@/layout";
import { create } from "@/lineplot/layout";
import { Workspace } from "@/workspace";

export interface UseCreateProps {
  workspace?: workspace.Key;
}

// useCreate creates a line plot on the server and places its layout once the
// create succeeds, so the plot's document exists before the renderer retrieves
// it. Mirrors the schematic create flow.
export const useCreate = ({
  workspace,
}: UseCreateProps): ((plot?: Partial<lineplot.New>) => void) => {
  const activeWorkspace = Workspace.useSelectActiveKey();
  const maybeChangeWorkspace = Workspace.useMaybeChange();
  const placeLayout = Layout.usePlacer();
  const { update } = LinePlot.useCreate({
    afterSuccess: async ({ data }) => {
      const { workspace, key, name } = data;
      if (workspace != null) await maybeChangeWorkspace(workspace);
      placeLayout(create({ key, name }));
    },
  });
  return useCallback(
    (plot) =>
      update({
        name: "Line Plot",
        ...plot,
        workspace: workspace ?? activeWorkspace ?? undefined,
      }),
    [workspace, activeWorkspace, maybeChangeWorkspace, placeLayout, update],
  );
};
