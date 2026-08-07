// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status, Synnax } from "@synnaxlabs/pluto";
import { strings } from "@synnaxlabs/x";
import { useCallback } from "react";

import { LinePlot } from "@/platform/lineplot";
import { Session } from "@/session";

export const useAddToNewPlot = (): ((keys: string[]) => void) => {
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();
  const create = LinePlot.useCreate();
  const dispatch = Session.useDispatch();
  return useCallback(
    (keys: string[]) => {
      if (client == null) return;
      handleError(async () => {
        const ranges = await client.ranges.retrieve({ keys });
        dispatch(Session.Range.add(Session.Range.fromClient(ranges)));
        const names = ranges.map(({ name }) => name);
        create({
          name: `Plot for ${strings.naturalLanguageJoin(names, "range")}`,
          ranges: { x1: ranges.map(({ key }) => key), x2: [] },
        });
      }, "Failed to add ranges to plot");
    },
    [client, dispatch, create, handleError],
  );
};
