// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot } from "@synnaxlabs/client";
import { LinePlot } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { useEffect, useRef } from "react";

import { Session } from "@/session";

export interface UseResetZoomOnRangeChangeProps {
  key: lineplot.Key;
}

/** Resets the plot's zoom in this window when the plot's ranges change. */
export const useResetZoomOnRangeChange = ({
  key,
}: UseResetZoomOnRangeChangeProps): void => {
  const ranges = LinePlot.useRanges({ key });
  const dispatch = Session.useDispatch();
  const shown = useRef(ranges);
  useEffect(() => {
    if (deep.equal(shown.current, ranges)) return;
    shown.current = ranges;
    dispatch(Session.LinePlot.setViewport({ key }));
  }, [ranges, dispatch, key]);
};
