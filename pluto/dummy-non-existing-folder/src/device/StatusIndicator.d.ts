import "./StatusIndicator.css";
import { type device } from "@synnaxlabs/client";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
export interface StatusIndicatorProps {
    status?: device.Status;
    tooltipLocation?: Tooltip.DialogProps["location"];
}
export declare const StatusIndicator: ({ status, tooltipLocation, }: StatusIndicatorProps) => import("react").JSX.Element;
//# sourceMappingURL=StatusIndicator.d.ts.map