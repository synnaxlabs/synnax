// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  LayoutStoreState,
  RemoveLayoutPayload,
  removeLayout,
  selectLayoutState,
  selectTheme,
} from "@/layout";
import { selectLinePlot, selectLineSliceState } from "@/line/store/selectors";
import {
  AddLinePlotYChannelPayload,
  CreateLinePlotPayload,
  DeleteLinePlotPayload,
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
} from "@/line/store/slice";
import { MiddlewareEffect, effectMiddleware } from "@/middleware";

export const assignColorsEffect: MiddlewareEffect<
  LayoutStoreState & LineStoreState,
  | CreateLinePlotPayload
  | SetLinePlotRangesPayload
  | SetLinePlotXChannelPayload
  | SetLinePlotYChannelsPayload
  | AddLinePlotYChannelPayload,
  SetLinePlotLinePaylaod
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
            color: colors[p.lines.indexOf(l) % colors.length].hex,
          },
        })
      );
    }
  });
};

export const deleteVisualizationEffect: MiddlewareEffect<
  LayoutStoreState & LineStoreState,
  RemoveLayoutPayload,
  DeleteLinePlotPayload
> = ({ action, dispatch, getState }) => {
  const state = getState();
  const vis = selectLineSliceState(state);
  const layout = selectLayoutState(state);
  Object.keys(vis.plots).forEach((key) => {
    if (layout.layouts[key] == null) dispatch(deleteLinePlot({ layoutKey: key }));
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
