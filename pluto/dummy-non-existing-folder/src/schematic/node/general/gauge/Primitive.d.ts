import "./gauge.css";
import { type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";
interface RenderProps extends Omit<schematic.GaugeNodeConfig, "variant" | "label" | "scale"> {
    className?: string;
}
export declare const Gauge: ({ color: c, className }: RenderProps) => ReactElement;
export {};
//# sourceMappingURL=Primitive.d.ts.map