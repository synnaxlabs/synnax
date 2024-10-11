import { Icon } from "@synnaxlabs/media";
import { Log as Core, telem } from "@synnaxlabs/pluto";
import { deep, id, TimeSpan } from "@synnaxlabs/x";

import { Layout } from "@/layout";
import { useSelect } from "@/log/selectors";
import { internalCreate, State, ZERO_STATE } from "@/log/slice";

export type LayoutType = "log";
export const LAYOUT_TYPE = "log";

export const Log: Layout.Renderer = ({ layoutKey }) => {
  const log = useSelect(layoutKey);
  const t = telem.streamChannelData({
    channel: log.channels[0] ?? 0,
    timeSpan: TimeSpan.seconds(5),
  });
  return <Core.Log telem={t} />;
};

export const SELECTABLE: Layout.Selectable = {
  key: LAYOUT_TYPE,
  title: "Log",
  icon: <Icon.Log />,
  create: (key) => create({ key }),
};

export const create =
  (initial: Partial<State> & Omit<Partial<Layout.State>, "type">): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Log", location = "mosaic", window, tab, ...rest } = initial;
    const key = initial.key ?? id.id();
    dispatch(
      internalCreate({
        ...deep.copy(ZERO_STATE),
        ...rest,
        key,
      }),
    );

    return {
      key,
      name,
      icon: "Log",
      location,
      type: LAYOUT_TYPE,
      windowKey: key,
      window,
      tab,
    };
  };
