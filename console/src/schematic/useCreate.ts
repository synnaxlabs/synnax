// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, type project, schematic } from "@synnaxlabs/client";
import { Schematic } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Layout } from "@/layout";
import { Project } from "@/project";
import { create } from "@/schematic/layout";

export interface UseCreateProps {
  project?: project.Key;
  // onResolved, when provided, replaces opening the schematic as a mosaic tab: the
  // created schematic's ontology.ID is handed back instead. The panel selector uses
  // this to fill a tab's resource.
  onResolved?: (resource: ontology.ID) => void;
}

// useCreate dispatches a schematic create against the active project (or an
// override). The workspace-era "switch projects on creation" behavior is gone;
// callers either operate on the active project or pass one explicitly.
export const useCreate = ({
  project,
  onResolved,
}: UseCreateProps): ((schematic?: Partial<schematic.Schematic>) => void) => {
  const activeProject = Project.useSelectActiveKey();
  const placeLayout = Layout.usePlacer();
  const { update } = Schematic.useCreate({
    afterSuccess: async ({ data }) => {
      const { key, name } = data;
      if (onResolved != null) onResolved(schematic.ontologyID(key));
      else placeLayout(create({ key, name, editable: true }));
    },
  });
  return useCallback(
    (schem) =>
      update({
        ...schematic.ZERO_NEW,
        name: "Schematic",
        ...schem,
        project: project ?? activeProject ?? undefined,
      }),
    [project, activeProject, update],
  );
};
