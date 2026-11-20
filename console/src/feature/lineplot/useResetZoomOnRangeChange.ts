// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot, type ranger } from "@synnaxlabs/client";
import { LinePlot } from "@synnaxlabs/pluto";
import { useEffect, useRef } from "react";

import { Session } from "@/session";

export interface UseResetZoomOnRangeChangeProps {
  key: lineplot.Key;
}

const sameRanges = (a: ranger.Ref[], b: ranger.Ref[]): boolean =>
  a.length === b.length && a.every((r, i) => Session.Range.refsEqual(r, b[i]));

/**
 * Resets the plot's zoom in this window when the plot shows other ranges. A change to
 * the T0 or color of a range keeps the zoom.
 */
export const useResetZoomOnRangeChange = ({
  key,
}: UseResetZoomOnRangeChangeProps): void => {
  const ranges = LinePlot.useRanges({ key });
  const dispatch = Session.useDispatch();
  const shown = useRef(ranges);
  useEffect(() => {
    if (sameRanges(shown.current, ranges)) return;
    shown.current = ranges;
    dispatch(Session.LinePlot.setViewport({ key }));
  }, [ranges, dispatch, key]);
};
