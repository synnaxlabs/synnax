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
import { type ReactElement, useCallback } from "react";

import { Range as PRange } from "@/platform/range";

export const Range = (): ReactElement => {
  const key = Log.useKey();
  const range = Log.useRange();
  const custom = Log.useCustomRange();
  const dispatch = Log.useSingleDispatch();
  const hasEditPermission = Access.useUpdateGranted(log.ontologyID(key));
  const handleRangeChange = useCallback(
    (range: string) => dispatch(log.setRange({ range })),
    [dispatch],
  );
  const handleCustomChange = useCallback(
    (custom: lineplot.CustomRange) => dispatch(log.setCustomRange({ custom })),
    [dispatch],
  );
  return (
    <Flex.Box x gap="large">
      <PRange.SelectSingleInputItem
        value={range}
        onChange={handleRangeChange}
        disabled={!hasEditPermission}
      />
      {range === PRange.CUSTOM_KEY && (
        <PRange.CustomInput x={false} value={custom} onChange={handleCustomChange} />
      )}
    </Flex.Box>
  );
};
