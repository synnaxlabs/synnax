import "./StatusIndicator.css";
import { type rack } from "@synnaxlabs/client";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
export interface StatusIndicatorProps {
    status?: rack.Status;
    tooltipLocation?: Tooltip.DialogProps["location"];
}
export declare const StatusIndicator: ({ status, tooltipLocation, }: StatusIndicatorProps) => import("react").JSX.Element;
//# sourceMappingURL=StatusIndicator.d.ts.map