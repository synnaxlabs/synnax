// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Video } from "@synnaxlabs/pluto";
import { resolveWindow, type TimeRange, TimeStamp } from "@synnaxlabs/x";

import { Session } from "@/session";

/**
 * Returns the time range the view plays. It is undefined when the view shows live
 * video, and null while the Core resolves the saved range of the view.
 */
export const useTimeRange = (): TimeRange | null | undefined => {
  const ref = Video.useRange();
  const [window] = Session.Range.useResolveWindows(ref == null ? [] : [ref]);
  if (ref != null && window == null) return null;
  // A rolling window ends now, so it shows live video.
  return window?.variant === "static"
    ? resolveWindow(window, TimeStamp.now())
    : undefined;
};
