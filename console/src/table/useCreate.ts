// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type project, table } from "@synnaxlabs/client";
import { Table as PTable } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Layout } from "@/layout";
import { Project } from "@/project";
import { type Selector } from "@/selector";
import { create } from "@/table/layout";

export interface UseCreateProps {
  project?: project.Key;
  // onResolved, when provided, replaces opening the table as a mosaic tab: the created
  // table is handed back as a resource to fill a panel tab in place.
  onResolved?: (content: Selector.ResolvedContent) => void;
}

export const useCreate = ({
  project,
  onResolved,
}: UseCreateProps): ((init?: Partial<table.Table>) => void) => {
  const activeProject = Project.useSelectActiveKey();
  const maybeChangeProject = Project.useMaybeChange();
  const placeLayout = Layout.usePlacer();
  const { update } = PTable.useCreate({
    afterSuccess: async ({ data }) => {
      const { project, key, name } = data;
      if (onResolved != null) {
        onResolved({ resource: table.ontologyID(key) });
        return;
      }
      if (project != null) await maybeChangeProject(project);
      placeLayout(create({ key, name, editable: true }));
    },
  });
  return useCallback(
    (init) =>
      update({
        name: "Table",
        ...init,
        project: project ?? activeProject ?? undefined,
      }),
    [project, activeProject, update],
  );
};
