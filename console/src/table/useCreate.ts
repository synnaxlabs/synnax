// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type table, type workspace } from "@synnaxlabs/client";
import { Table as PTable } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Layout } from "@/layout";
import { create } from "@/table/layout";
import { Workspace } from "@/workspace";

export interface UseCreateProps {
  workspace?: workspace.Key;
}

export const useCreate = ({
  workspace,
}: UseCreateProps): ((init?: Partial<table.Table>) => void) => {
  const activeWorkspace = Workspace.useSelectActiveKey();
  const maybeChangeWorkspace = Workspace.useMaybeChange();
  const placeLayout = Layout.usePlacer();
  const { update } = PTable.useCreate({
    afterSuccess: async ({ data }) => {
      const { workspace, key, name } = data;
      if (workspace != null) await maybeChangeWorkspace(workspace);
      placeLayout(create({ key, name, editable: true }));
    },
  });
  return useCallback(
    (init) =>
      update({
        name: "Table",
        ...init,
        workspace: workspace ?? activeWorkspace ?? undefined,
      }),
    [workspace, activeWorkspace, update],
  );
};
