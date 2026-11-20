// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";
import { strings } from "@synnaxlabs/x";

import { LinePlot } from "@/platform/lineplot";

/**
 * @returns a function that opens a new line plot of the ranges with the given keys. A
 * relative time mode draws each range from its start, to compare the ranges.
 */
export const useAddToNewPlot = (): ((
  keys: string[],
  timeMode?: lineplot.TimeMode,
) => void) => {
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();
  const create = LinePlot.useCreate();
  return (keys, timeMode) => {
    if (client == null) return;
    const relative = timeMode === "relative";
    handleError(
      async () => {
        const ranges = await client.ranges.retrieve(keys);
        const names = strings.naturalLanguageJoin(
          ranges.map(({ name }) => name),
          "range",
        );
        create({
          name: relative ? `Compare ${names}` : `Plot for ${names}`,
          ranges: ranges.map(({ key }) => ({ variant: "key", key })),
          axes: { x1: { key: "x1", timeMode } },
        });
      },
      relative ? "Failed to compare ranges" : "Failed to add ranges to plot",
    );
  };
};
