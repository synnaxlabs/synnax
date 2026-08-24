// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Session } from "@/session";

export const useAddToActivePlot = (): ((keys: string[]) => void) => {
  const handleError = Status.useErrorHandler();
  const dispatch = Session.useDispatch();
  const client = Synnax.use();
  const getFocusedKey = Session.LinePlot.useGetFocusedKey();
  return useCallback(
    (keys: string[]) => {
      const active = getFocusedKey();
      if (active == null || client == null) return;
      handleError(async () => {
        const ranges = await client.ranges.retrieve({ keys });
        dispatch(Session.Range.add(Session.Range.fromClient(ranges)));
        await client.lineplots.dispatch(
          active,
          ranges.map((range) => lineplot.addRange({ axisKey: "x1", range: range.key })),
        );
      }, "Failed to add ranges to plot");
    },
    [client, dispatch, getFocusedKey, handleError],
  );
};
