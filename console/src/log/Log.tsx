import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Icon } from "@synnaxlabs/media";
import { Log as Core, telem } from "@synnaxlabs/pluto";
import { deep, id, TimeSpan } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import { useSelect } from "@/log/selectors";
import { internalCreate, State, ZERO_STATE } from "@/log/slice";

export type LayoutType = "log";
export const LAYOUT_TYPE = "log";

export const Log: Layout.Renderer = ({ layoutKey, visible }) => {
  const winKey = useSelectWindowKey() as string;
  const log = useSelect(layoutKey);
  const dispatch = useDispatch();
  let t: telem.SeriesSourceSpec;
  if (log.channels[0] == null || log.channels[0] === 0) t = telem.noopSeriesSourceSpec;
  else
    t = telem.streamChannelData({
      channel: log.channels[0] ?? 0,
      timeSpan: TimeSpan.seconds(log.retention),
    });

  const handleDoubleClick = useCallback(() => {
    dispatch(
      Layout.setNavDrawerVisible({
        windowKey: winKey,
        key: "visualization",
        value: true,
      }),
    );
  }, [winKey, dispatch]);

  return <Core.Log telem={t} onDoubleClick={handleDoubleClick} visible={visible} />;
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
    dispatch(internalCreate({ ...deep.copy(ZERO_STATE), ...rest, key }));
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
