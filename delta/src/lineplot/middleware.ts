// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Layout } from "@/layout";
import { select, selectSliceState } from "@/lineplot/selectors";
import {
  type AddYChannelPayload,
  type CreatePayload,
  type RemovePayload,
  type StoreState,
  type SetLinePayload,
  type SetRangesPayload,
  type SetXChannelPayload,
  type SetYChannelsPayload,
  actions,
  remove,
  setLine,
  setRanges,
  setXChannel,
  setYChannels,
} from "@/lineplot/slice";
import { type MiddlewareEffect, effectMiddleware } from "@/middleware";

export const assignColorsEffect: MiddlewareEffect<
  Layout.StoreState & StoreState,
  | CreatePayload
  | SetRangesPayload
  | SetXChannelPayload
  | SetYChannelsPayload
  | AddYChannelPayload,
  SetLinePayload
> = ({ getState, action, dispatch }) => {
  const s = getState();
  const p = select(s, action.payload.key);
  p.lines.forEach((l) => {
    if (l.color === "") {
      const theme = Layout.selectTheme(s);
      const colors = theme?.colors.visualization.palettes.default ?? [];
      dispatch(
        setLine({
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

export const deleteEffect: MiddlewareEffect<
  Layout.StoreState & StoreState,
  Layout.RemovePayload | Layout.SetSlicePayload,
  RemovePayload
> = ({ action, dispatch, getState }) => {
  const state = getState();
  const lineState = selectSliceState(state);
  const layout = Layout.selectSliceState(state);
  const keys = "keys" in action.payload ? action.payload.keys : [];
  const toRemove = Object.keys(lineState.plots).filter(
    (p) => layout.layouts[p] == null || keys.includes(p)
  );
  if (toRemove.length > 0) dispatch(remove({ layoutKeys: toRemove }));
};

export const MIDDLEWARE = [
  effectMiddleware(
    [
      actions.set.type,
      setXChannel.type,
      setYChannels.type,
      setRanges.type,
      setYChannels.type,
    ],
    [assignColorsEffect]
  ),
  effectMiddleware([Layout.remove.type, Layout.setWorkspace.type], [deleteEffect]),
];
