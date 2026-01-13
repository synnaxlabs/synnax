import { arc } from "@synnaxlabs/client";
import { Access, Arc as Core, Icon } from "@synnaxlabs/pluto";
import { deep, uuid } from "@synnaxlabs/x";

import { useSelectVersion } from "@/arc/selectors";
import { internalCreate, type State } from "@/arc/slice";
import { ZERO_STATE } from "@/arc/types";
import { translateGraphToConsole } from "@/arc/types/translate";
import { TYPE } from "@/arc/types/v0";
import { createLoadRemote } from "@/hooks/useLoadRemote";
import { type Layout } from "@/layout";
import { type Selector } from "@/selector";

export const LAYOUT_TYPE = "arc_editor";
export type LayoutType = typeof LAYOUT_TYPE;

export const SELECTABLE: Selector.Selectable = {
  key: LAYOUT_TYPE,
  title: "Arc Automation",
  icon: <Icon.Arc />,
  useVisible: () => Access.useUpdateGranted(arc.TYPE_ONTOLOGY_ID),
  create: async ({ layoutKey, rename }) => {
    const name = await rename({}, { icon: "Arc", name: "Arc.Create" });
    if (name == null) return null;
    return create({ key: layoutKey, name });
  },
};

export type CreateArg = Partial<State> & Partial<Layout.BaseState>;

export const create =
  (initial: CreateArg = {}): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Arc Editor", location = "mosaic", window, tab, ...rest } = initial;
    const key = arc.keyZ.safeParse(initial.key).data ?? uuid.create();
    dispatch(internalCreate({ ...deep.copy(ZERO_STATE), ...rest, key }));
    return {
      key,
      location,
      name,
      icon: "Arc",
      type: LAYOUT_TYPE,
      window: { navTop: true, showTitle: true },
      tab,
    };
  };

export const useLoadRemote = createLoadRemote<arc.Arc>({
  useRetrieve: Core.useRetrieveObservable,
  targetVersion: ZERO_STATE.version,
  useSelectVersion,
  actionCreator: (v) =>
    internalCreate({
      version: "0.0.0",
      key: v.key,
      type: TYPE,
      remoteCreated: false,
      graph: translateGraphToConsole(v.graph),
      text: { raw: "" },
      mode: "graph",
    }),
});
