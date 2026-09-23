import "./tank.css";
import { type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";
interface RenderProps extends Partial<Pick<schematic.TankNodeConfig, "dimensions" | "borderRadius" | "color" | "backgroundColor">> {
    className?: string;
    boxBorderRadius?: number;
    strokeWidth?: number;
}
export declare const Tank: ({ className, dimensions, borderRadius, boxBorderRadius, color: colorVal, backgroundColor, strokeWidth, }: RenderProps) => ReactElement;
export {};
//# sourceMappingURL=Primitive.d.ts.map