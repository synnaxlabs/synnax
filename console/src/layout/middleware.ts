// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { Mosaic } from "@synnaxlabs/pluto";

import { selectSliceState } from "@/layout/selectors";
import {
  moveMosaicTab,
  type MoveMosaicTabPayload,
  remove,
  type RemovePayload,
  type StoreState,
} from "@/layout/slice";
import { effectMiddleware, type MiddlewareEffect } from "@/middleware";

export const closeWindowOnEmptyMosaicEffect: MiddlewareEffect<
  StoreState & Drift.StoreState,
  MoveMosaicTabPayload | RemovePayload,
  Drift.CloseWindowPayload
> = ({ getState, dispatch }) => {
  const s = getState();
  const { mosaics } = selectSliceState(s);
  Object.entries(mosaics).forEach(([k, { root }]) => {
    if (k !== Drift.MAIN_WINDOW && Mosaic.isEmpty(root)) {
      const win = Drift.selectWindow(s, k);
      if (win != null) dispatch(Drift.closeWindow({ key: k }));
    }
  });
};

export const MIDDLEWARE = [
  effectMiddleware([moveMosaicTab.type, remove.type], [closeWindowOnEmptyMosaicEffect]),
];
