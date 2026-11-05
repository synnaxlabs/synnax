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

const WORDS: Record<Status.Variant, string> = {
  success: "Connected",
  info: "Info",
  warning: "Warning",
  error: "Error",
  loading: "Loading",
  disabled: "Unknown",
};

export interface StatusChipProps {
  status?: device.Status;
  tooltipLocation?: Tooltip.DialogProps["location"];
}

/** Names a device's status in one word. A tooltip shows the full message. */
export const StatusChip = ({
  status,
  tooltipLocation = "bottom",
}: StatusChipProps): ReactElement => {
  const variant = status?.variant ?? "disabled";
  const message = status?.message ?? "Device status unknown";
  return (
    <Tooltip.Dialog location={tooltipLocation}>
      <Status.Summary variant={variant} hideIcon level="small" weight={450}>
        {message}
      </Status.Summary>
      <Status.Chip variant={variant}>{WORDS[variant]}</Status.Chip>
    </Tooltip.Dialog>
  );
};
