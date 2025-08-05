import "@/hardware/device/StatusIndicator.css";

import { type device } from "@synnaxlabs/client";

import { CSS } from "@/css";
import { Status } from "@/status";
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
