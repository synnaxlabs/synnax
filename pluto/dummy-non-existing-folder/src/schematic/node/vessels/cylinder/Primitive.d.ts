import { type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";
interface RenderProps extends Partial<Pick<schematic.CylinderNodeConfig, "dimensions" | "borderRadius" | "color" | "backgroundColor" | "orientation" | "scale">> {
    className?: string;
}
export declare const Cylinder: ({ className, dimensions, borderRadius, color: colorVal, backgroundColor, orientation, scale, }: RenderProps) => ReactElement;
export {};
//# sourceMappingURL=Primitive.d.ts.map