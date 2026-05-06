import { schematic } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";

import { type Layout } from "@/layout";
import { internalCreate } from "@/schematic/slice";

export const LAYOUT_TYPE = "schematic";
export type LayoutType = typeof LAYOUT_TYPE;

export interface CreateArg extends Partial<Layout.BaseState> {
  key?: string;
}

export const create =
  (initial: CreateArg = {}): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Schematic", location = "mosaic", tab } = initial;
    const key = schematic.keyZ.safeParse(initial.key).data ?? uuid.create();
    dispatch(internalCreate({ key }));
    return {
      key,
      location,
      name,
      icon: "Schematic",
      type: LAYOUT_TYPE,
      window: { navTop: true, showTitle: true },
      tab,
    };
  };
