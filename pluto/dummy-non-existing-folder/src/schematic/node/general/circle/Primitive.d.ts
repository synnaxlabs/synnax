import { type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";
interface RenderProps extends Omit<schematic.CircleNodeConfig, "variant" | "label" | "scale"> {
    className?: string;
}
export declare const Circle: ({ radius, color: colorVal, backgroundColor, className, strokeWidth, }: RenderProps) => ReactElement;
export {};
//# sourceMappingURL=Primitive.d.ts.map