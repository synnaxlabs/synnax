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
import { type bounds } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { CSS } from "@/css";

const RECEIPT_TIMESTAMP_PRECISION_BOUNDS: bounds.Bounds = { lower: 0, upper: 4 };

export const Properties = (): ReactElement => {
  const key = Log.useKey();
  const dispatch = Log.useSingleDispatch();
  const hideChannelNames = Log.useSelectHideChannelNames();
  const hideReceiptTimestamp = Log.useSelectHideReceiptTimestamp();
  const timestampPrecision = Log.useSelectTimestampPrecision();
  const hasEditPermission = Access.useUpdateGranted(log.ontologyID(key));

  const handlePrecisionChange = useCallback(
    (timestampPrecision: number) =>
      dispatch(log.setTimestampPrecision({ timestampPrecision })),
    [],
  );

  const handleShowChannelNamesChange = useCallback(
    (visible: boolean) =>
      dispatch(log.setHideChannelNames({ hideChannelNames: !visible })),
    [dispatch],
  );

  const handleShowReceiptTimestampChange = useCallback(
    (visible: boolean) =>
      dispatch(log.setHideReceiptTimestamp({ hideReceiptTimestamp: !visible })),
    [dispatch],
  );

  return (
    <Flex.Box x className={CSS.BE("log", "toolbar", "properties")}>
      <Input.Item label="Show Receipt Timestamp">
        <Input.Switch
          value={!hideReceiptTimestamp}
          onChange={handleShowReceiptTimestampChange}
          disabled={!hasEditPermission}
        />
      </Input.Item>
      <Input.Item label="Receipt Timestamp Precision">
        <Input.Numeric
          value={timestampPrecision}
          onChange={handlePrecisionChange}
          resetValue={0}
          bounds={RECEIPT_TIMESTAMP_PRECISION_BOUNDS}
          disabled={!hasEditPermission}
        />
      </Input.Item>
      <Input.Item label="Show Channel Names">
        <Input.Switch
          value={!hideChannelNames}
          onChange={handleShowChannelNamesChange}
          disabled={!hasEditPermission}
        />
      </Input.Item>
    </Flex.Box>
  );
};
