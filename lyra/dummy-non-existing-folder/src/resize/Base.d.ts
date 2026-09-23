import "./Base.css";
import { location } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { Flex } from "../flex";
export interface BaseProps extends Omit<Flex.BoxProps, "gap" | "size" | "direction" | "x" | "y"> {
    location: location.Crude;
    size: number;
    decimal?: boolean;
    hideHandle?: boolean;
    /** Drops the drag handle while keeping the divider, for a pane that cannot resize. */
    disabled?: boolean;
}
export declare const Base: ({ ref, location: cloc, style: propsStyle, size, className, children, onPointerDown, decimal, hideHandle, disabled, ...rest }: BaseProps) => ReactElement;
//# sourceMappingURL=Base.d.ts.map