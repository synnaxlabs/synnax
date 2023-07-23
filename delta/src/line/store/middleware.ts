import { selectLinePlot, selectLineSliceState } from "./selectors";
import {
  AddLinePlotYChannelPayload,
  CreateLinePlotPayload,
  LineStoreState,
  SetLinePlotLinePaylaod,
  SetLinePlotRangesPayload,
  SetLinePlotXChannelPayload,
  SetLinePlotYChannelsPayload,
  actions,
  addLinePlotYChannel,
  deleteLinePlot,
  setLinePlotLine,
  setLinePlotRanges,
  setLinePlotXChannel,
  setLinePlotYChannels,
} from "./slice";

import {
  LayoutStoreState,
  RemoveLayoutPayload,
  removeLayout,
  selectLayoutState,
  selectTheme,
} from "@/layout";
import { MiddlewareEffect, effectMiddleware } from "@/middleware";

export const assignColorsEffect: MiddlewareEffect<
  LayoutStoreState & LineStoreState,
  | CreateLinePlotPayload
  | SetLinePlotRangesPayload
  | SetLinePlotXChannelPayload
  | SetLinePlotYChannelsPayload
  | SetLinePlotLinePaylaod
  | AddLinePlotYChannelPayload
> = ({ getState, action, dispatch }) => {
  const s = getState();
  const p = selectLinePlot(s, action.payload.key);
  p.lines.forEach((l) => {
    if (l.color === "") {
      const theme = selectTheme(s);
      const colors = theme?.colors.visualization.palettes.default ?? [];
      dispatch(
        setLinePlotLine({
          key: p.key,
          line: {
            key: l.key,
            color: colors[p.lines.indexOf(l) % colors.length] as string,
          },
        })
      );
    }
  });
};

export const deleteVisualizationEffect: MiddlewareEffect<
  LayoutStoreState & LineStoreState,
  RemoveLayoutPayload
> = ({ action, dispatch, getState }) => {
  const state = getState();
  const vis = selectLineSliceState(state);
  const layout = selectLayoutState(state);
  Object.keys(vis.plots).forEach((key) => {
    if (layout.layouts[key] == null) {
      dispatch(deleteLinePlot({ layoutKey: key }));
    }
  });
  const p = selectLinePlot(getState(), action.payload);
  if (p != null) dispatch(deleteLinePlot({ layoutKey: action.payload }));
};

export const lineMiddleware = [
  effectMiddleware(
    [
      actions.setLinePlot.type,
      setLinePlotXChannel.type,
      setLinePlotYChannels.type,
      setLinePlotRanges.type,
      addLinePlotYChannel.type,
    ],
    [assignColorsEffect]
  ),
  effectMiddleware([removeLayout.type], [deleteVisualizationEffect]),
];
