// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { device } from "@synnaxlabs/client";
import { Badge } from "@synnaxlabs/lyra/badge";
import { Status } from "@synnaxlabs/lyra/status";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import type { ReactElement } from "react";

export interface StatusBadgeProps {
  status?: device.Status;
  tooltipLocation?: Tooltip.DialogProps["location"];
}

/**
 * Says whether the Driver can reach a device, in the color of its status. A tooltip
 * shows the status message and description. A device with no status, or with the
 * disabled status that the Core gives a device no Driver has reached, shows a gray
 * "Unknown" badge with no tooltip.
 */
export const StatusBadge = ({
  status,
  tooltipLocation = "bottom",
}: StatusBadgeProps): ReactElement => {
  if (status == null || status.variant === "disabled")
    return <Badge.Badge>Unknown</Badge.Badge>;
  return (
    <Tooltip.Dialog location={tooltipLocation}>
      <Status.Summary
        variant={status.variant}
        message={status.message}
        description={status.description}
        hideIcon
        level="small"
      />
      <Badge.Badge status={status.variant}>
        {status.details.connected ? "Connected" : "Disconnected"}
      </Badge.Badge>
    </Tooltip.Dialog>
  );
};
