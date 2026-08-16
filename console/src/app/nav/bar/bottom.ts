// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useMemo } from "react";

import { Session } from "@/session";

export interface BottomActions {
  select: () => void;
  toggle: () => void;
  pin: () => void;
  startHover: () => void;
  stopHover: () => void;
}

/** Bound dispatchers for the bottom drawer's toolbar item. */
export const useBottomActions = (): BottomActions => {
  const dispatch = Session.useDispatch();
  return useMemo(
    () => ({
      select: () => dispatch(Session.Nav.selectBottom({})),
      toggle: () => dispatch(Session.Nav.toggleBottom({})),
      pin: () => dispatch(Session.Nav.showBottom({})),
      startHover: () => dispatch(Session.Nav.startBottomHover({})),
      stopHover: () => dispatch(Session.Nav.stopBottomHover({})),
    }),
    [dispatch],
  );
};
