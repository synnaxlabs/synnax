// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { type ReactElement } from "react";

export interface StatusBadgeProps {
  status?: device.Status;
  tooltipLocation?: Tooltip.DialogProps["location"];
}

/**
 * Says whether the Driver can reach a device, in the color of its status. A tooltip
 * shows the status message and description.
 */
export const StatusBadge = ({
  status,
  tooltipLocation = "bottom",
}: StatusBadgeProps): ReactElement => {
  const variant = status?.variant ?? "disabled";
  return (
    <Tooltip.Dialog location={tooltipLocation}>
      <Status.Summary
        variant={variant}
        message={status?.message ?? "Device status unknown"}
        description={status?.description}
        hideIcon
        level="small"
      />
      <Status.Chip variant={variant}>
        {status?.details.connected === true ? "Connected" : "Disconnected"}
      </Status.Chip>
    </Tooltip.Dialog>
  );
};
