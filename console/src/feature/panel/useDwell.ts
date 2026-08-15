// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/pluto";
import { TimeSpan } from "@synnaxlabs/x";
import { useCallback, useEffect, useMemo, useRef } from "react";
import { useDispatch } from "react-redux";

import { Session } from "@/session";

// How long a tab must rest on a pill before the strip switches to that panel. Long
// enough that dragging across the strip to a far pill switches nothing on the way.
const DWELL_DURATION = TimeSpan.milliseconds(450).milliseconds;

export interface Dwell {
  /** Starts the switch timer for a panel. Call from a pill's drag-over. */
  enter: (key: panel.Key) => void;
  /** Cancels a pending switch for a panel. Call from a pill's drag-leave. */
  leave: (key: panel.Key) => void;
}

/**
 * useDwell switches the visible panel when a dragged tab rests on a pill, so the drag
 * finishes as an ordinary drop inside the destination's mosaic. A drag that ends
 * without dropping snaps back to the panel selected before the first switch. Mount
 * once per strip.
 */
export const useDwell = (): Dwell => {
  const dispatch = useDispatch();
  const getSelected = Session.Panel.useGetSelected();
  const timeout = useRef<ReturnType<typeof setTimeout>>(undefined);
  const pending = useRef<panel.Key>(undefined);
  // The panel to return to, set on the first switch of a drag.
  const origin = useRef<panel.Key>(undefined);

  const cancel = useCallback(() => {
    clearTimeout(timeout.current);
    timeout.current = undefined;
    pending.current = undefined;
  }, []);

  const enter = useCallback(
    (key: panel.Key) => {
      if (pending.current === key) return;
      cancel();
      pending.current = key;
      timeout.current = setTimeout(() => {
        pending.current = undefined;
        const selected = getSelected();
        if (selected === key) return;
        origin.current ??= selected;
        dispatch(Session.Panel.select({ key }));
      }, DWELL_DURATION);
    },
    [cancel, dispatch, getSelected],
  );

  const leave = useCallback(
    (key: panel.Key) => {
      // Moving between neighboring pills can report the leave after the next pill's
      // drag-over, which must not cancel the timer that just started.
      if (pending.current === key) cancel();
    },
    [cancel],
  );

  Haul.useOnResolve((landed) => {
    cancel();
    const key = origin.current;
    origin.current = undefined;
    if (!landed && key != null) dispatch(Session.Panel.select({ key }));
  });

  useEffect(() => () => clearTimeout(timeout.current), []);

  return useMemo(() => ({ enter, leave }), [enter, leave]);
};
