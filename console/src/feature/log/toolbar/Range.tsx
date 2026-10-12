// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot, log } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Access, Log } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useState } from "react";

import { Range as PRange } from "@/platform/range";

export const Range = (): ReactElement => {
  const key = Log.useKey();
  const range = Log.useRange();
  const custom = Log.useCustomRange();
  const dispatch = Log.useSingleDispatch();
  const hasEditPermission = Access.useUpdateGranted(log.ontologyID(key));
  // Focus the custom input only when the user picks it, not when the toolbar opens.
  const [customPicked, setCustomPicked] = useState(false);
  const handleRangeChange = useCallback(
    (range: string) => {
      setCustomPicked(range === PRange.CUSTOM_KEY);
      dispatch(log.setRange({ range }));
    },
    [dispatch],
  );
  const handleCustomChange = useCallback(
    (custom: lineplot.CustomRange) => dispatch(log.setCustomRange({ custom })),
    [dispatch],
  );
  return (
    <Flex.Box y gap="small">
      <PRange.SelectSingleInputItem
        value={range}
        onChange={handleRangeChange}
        disabled={!hasEditPermission}
      />
      {range === PRange.CUSTOM_KEY && (
        <PRange.CustomInput
          x={false}
          value={custom}
          onChange={handleCustomChange}
          autoFocus={customPicked}
        />
      )}
    </Flex.Box>
  );
};
