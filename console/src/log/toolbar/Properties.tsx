// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { Access, Flex, Input, Log } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

export interface PropertiesProps {
  layoutKey: string;
}

export const Properties = ({ layoutKey }: PropertiesProps): ReactElement => {
  const { dispatch } = Log.useDispatch();
  const showChannelNames = Log.useSelectShowChannelNames({ key: layoutKey });
  const showReceiptTimestamp = Log.useSelectShowReceiptTimestamp({ key: layoutKey });
  const timestampPrecision = Log.useSelectTimestampPrecision({ key: layoutKey });
  const hasEditPermission = Access.useUpdateGranted(log.ontologyID(layoutKey));

  const apply = useCallback(
    (action: log.Action) => dispatch({ key: layoutKey, actions: [action] }),
    [dispatch, layoutKey],
  );

  const handlePrecisionChange = useCallback(
    (v: number) => apply(log.setTimestampPrecision({ timestampPrecision: v })),
    [apply],
  );

  const handleShowChannelNamesChange = useCallback(
    (v: boolean) => apply(log.setShowChannelNames({ showChannelNames: v })),
    [apply],
  );

  const handleShowReceiptTimestampChange = useCallback(
    (v: boolean) => apply(log.setShowReceiptTimestamp({ showReceiptTimestamp: v })),
    [apply],
  );

  return (
    <Flex.Box x className="console-log__toolbar-properties">
      <Input.Item label="Show Receipt Timestamp">
        <Input.Switch
          value={showReceiptTimestamp}
          onChange={handleShowReceiptTimestampChange}
          disabled={!hasEditPermission}
        />
      </Input.Item>
      <Input.Item label="Receipt Timestamp Precision">
        <Input.Numeric
          value={timestampPrecision}
          onChange={handlePrecisionChange}
          resetValue={0}
          bounds={{ lower: 0, upper: 4 }}
          disabled={!hasEditPermission}
        />
      </Input.Item>
      <Input.Item label="Show Channel Names">
        <Input.Switch
          value={showChannelNames}
          onChange={handleShowChannelNamesChange}
          disabled={!hasEditPermission}
        />
      </Input.Item>
    </Flex.Box>
  );
};
