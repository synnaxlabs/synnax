import "./value.css";
import { type schematic } from "@synnaxlabs/client";
import { type dimensions, type text } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";
interface RenderProps extends PropsWithChildren<Pick<schematic.ValueNodeConfig, "color" | "orientation" | "units" | "inlineSize">> {
    className?: string;
    dimensions?: dimensions.Dimensions;
    unitsLevel?: text.Level;
}
export declare const Value: ({ className, color: colorVal, dimensions, orientation, units, unitsLevel, children, inlineSize, }: RenderProps) => ReactElement;
export {};
//# sourceMappingURL=Primitive.d.ts.map