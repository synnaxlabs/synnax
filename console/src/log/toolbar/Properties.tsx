// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { Access, Flex, Input } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { useSyncComponent } from "@/log/Log";
import { useSelectOptional } from "@/log/selectors";
import {
  setShowChannelNames,
  setShowReceiptTimestamp,
  setTimestampPrecision,
} from "@/log/slice";

export interface PropertiesProps {
  layoutKey: string;
}

export const Properties = ({ layoutKey }: PropertiesProps): ReactElement | null => {
  const dispatch = useSyncComponent(layoutKey);
  const state = useSelectOptional(layoutKey);
  const hasEditPermission = Access.useUpdateGranted(log.ontologyID(layoutKey));

  const handlePrecisionChange = useCallback(
    (v: number) =>
      dispatch(setTimestampPrecision({ key: layoutKey, timestampPrecision: v })),
    [dispatch, layoutKey],
  );

  const handleShowChannelNamesChange = useCallback(
    (v: boolean) =>
      dispatch(setShowChannelNames({ key: layoutKey, showChannelNames: v })),
    [dispatch, layoutKey],
  );

  const handleShowReceiptTimestampChange = useCallback(
    (v: boolean) =>
      dispatch(setShowReceiptTimestamp({ key: layoutKey, showReceiptTimestamp: v })),
    [dispatch, layoutKey],
  );

  if (state == null) return null;
  return (
    <Flex.Box x className="console-log__toolbar-properties">
      <Input.Item label="Show Receipt Timestamp">
        <Input.Switch
          value={state.showReceiptTimestamp}
          onChange={handleShowReceiptTimestampChange}
          disabled={!hasEditPermission}
        />
      </Input.Item>
      <Input.Item label="Receipt Timestamp Precision">
        <Input.Numeric
          value={state.timestampPrecision}
          onChange={handlePrecisionChange}
          resetValue={0}
          bounds={{ lower: 0, upper: 4 }}
          disabled={!hasEditPermission}
        />
      </Input.Item>
      <Input.Item label="Show Channel Names">
        <Input.Switch
          value={state.showChannelNames}
          onChange={handleShowChannelNamesChange}
          disabled={!hasEditPermission}
        />
      </Input.Item>
    </Flex.Box>
  );
};
