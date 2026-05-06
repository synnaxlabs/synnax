import { schematic, type workspace } from "@synnaxlabs/client";
import { Schematic } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Layout } from "@/layout";
import { create } from "@/schematic/layout";
import { Workspace } from "@/workspace";

export interface UseCreateProps {
  workspace?: workspace.Key;
}

export const useCreate = ({
  workspace,
}: UseCreateProps): ((schematic?: Partial<schematic.Schematic>) => void) => {
  const activeWorkspace = Workspace.useSelectActiveKey();
  const maybeChangeWorkspace = Workspace.useMaybeChange();
  const placeLayout = Layout.usePlacer();
  const { update } = Schematic.useCreate({
    afterSuccess: async ({ data }) => {
      const { workspace, key, name } = data;
      if (workspace != null) await maybeChangeWorkspace(workspace);
      placeLayout(create({ key, name }));
    },
  });
  return useCallback(
    (schem) =>
      update({
        ...schematic.ZERO_NEW,
        name: "New Schematic",
        ...schem,
        workspace: workspace ?? activeWorkspace ?? undefined,
      }),
    [workspace],
  );
};
