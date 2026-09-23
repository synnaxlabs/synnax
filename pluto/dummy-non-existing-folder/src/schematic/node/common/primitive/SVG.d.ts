import { dimensions } from "@synnaxlabs/x";
import { type ComponentPropsWithoutRef, type ReactElement } from "react";
import { type SVGBasedProps } from "./orientable";
export interface SVGProps extends SVGBasedProps, Omit<ComponentPropsWithoutRef<"svg">, "direction" | "color" | "orientation" | "scale"> {
    dimensions: dimensions.Dimensions;
}
export declare const BASE_SCALE = 0.8;
/** True inside a delayed toggle, so the SVG draws the hold fill within its shapes. */
export declare const HoldFill: import("react").Context<boolean>;
export declare const SVG: ({ dimensions: dimsProp, orientation, children, className, color: colorVal, style, scale, ...rest }: SVGProps) => ReactElement;
//# sourceMappingURL=SVG.d.ts.map