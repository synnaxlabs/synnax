import { selectLinePlot } from "./selectors";
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
  setLinePlotLine,
  setLinePlotRanges,
  setLinePlotXChannel,
  setLinePlotYChannels,
} from "./slice";

import { LayoutStoreState, selectTheme } from "@/layout";
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

export const lineMiddleware = [
  effectMiddleware(
    [
      actions.createLinePlot.type,
      setLinePlotXChannel.type,
      setLinePlotYChannels.type,
      setLinePlotRanges.type,
      addLinePlotYChannel.type,
    ],
    [assignColorsEffect]
  ),
];
