// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/device/StatusIndicator.css";

import { type device } from "@synnaxlabs/client";

import { CSS } from "@/css";
import { Status } from "@/status/base";
import { Tooltip } from "@/tooltip";

export interface StatusIndicatorProps {
  status?: device.Status;
  tooltipLocation?: Tooltip.DialogProps["location"];
}

export const StatusIndicator = ({
  status,
  tooltipLocation = "right",
}: StatusIndicatorProps) => {
  const variant = status?.variant ?? "disabled";
  const message = status?.message ?? "Device Status Unknown";
  return (
    <Tooltip.Dialog location={tooltipLocation}>
      <Status.Summary variant={variant} hideIcon level="small" weight={450}>
        {message}
      </Status.Summary>
      <Status.Indicator
        variant={variant}
        className={CSS.BE("device", "status-indicator")}
      />
    </Tooltip.Dialog>
  );
};
