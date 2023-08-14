// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DriftStoreState,
  CloseWindowPayload,
  closeWindow,
  MAIN_WINDOW,
  selectWindow,
} from "@synnaxlabs/drift";
import { Mosaic } from "@synnaxlabs/pluto";

import { selectLayoutState } from "@/layout/store/selectors";
import {
  LayoutStoreState,
  MoveLayoutMosaicTabPayload,
  RemoveLayoutPayload,
  moveLayoutMosaicTab,
  removeLayout,
} from "@/layout/store/slice";
import { effectMiddleware, MiddlewareEffect } from "@/middleware";

export const closeWindowOnEmptyMosaicEffect: MiddlewareEffect<
  LayoutStoreState & DriftStoreState,
  MoveLayoutMosaicTabPayload | RemoveLayoutPayload,
  CloseWindowPayload
> = ({ getState, action, dispatch }) => {
  const s = getState();
  const { mosaics } = selectLayoutState(s);
  Object.entries(mosaics).forEach(([k, { root }]) => {
    if (k !== MAIN_WINDOW && Mosaic.isEmpty(root)) {
      const win = selectWindow(s, k);
      if (win != null) dispatch(closeWindow({ key: k }));
    }
  });
};

export const layoutMiddleware = [
  effectMiddleware(
    [moveLayoutMosaicTab.type, removeLayout.type],
    [closeWindowOnEmptyMosaicEffect]
  ),
];
