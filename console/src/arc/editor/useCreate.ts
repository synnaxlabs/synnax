import { type arc } from "@synnaxlabs/client";
import { Arc, type Flux } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { create } from "@/arc/editor/layout";
import { ZERO_GRAPH } from "@/arc/slice";
import { Layout } from "@/layout";

interface CreateArgs {
  key?: string;
  name?: string;
  mode?: arc.Mode;
}

export const useCreate = (): ((args?: CreateArgs) => void) => {
  const placeLayout = Layout.usePlacer();
  const { update } = Arc.useCreate({
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
