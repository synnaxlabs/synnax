import { type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";
export declare const DEFAULT_POLYGON_SIDE_LENGTH = 20;
interface RenderProps extends Omit<schematic.PolygonNodeConfig, "variant" | "label" | "scale"> {
    className?: string;
}
export declare const Polygon: ({ numSides, sideLength, rotation, color: colorVal, backgroundColor, className, cornerRounding, strokeWidth, }: RenderProps) => ReactElement;
export {};
//# sourceMappingURL=Primitive.d.ts.map