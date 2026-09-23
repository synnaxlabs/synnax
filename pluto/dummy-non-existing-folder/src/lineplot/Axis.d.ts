import "./Axis.css";
import { Flex } from "@synnaxlabs/lyra/flex";
import { direction, type text } from "@synnaxlabs/x";
import { type FC, type PropsWithChildren } from "react";
import { type z } from "zod";
import { Aether } from "../aether";
import { lineplot } from "./aether";
export interface AxisProps extends PropsWithChildren, Omit<z.input<typeof lineplot.xAxisStateZ>, "position" | "size">, Omit<Flex.BoxProps, "color">, Aether.ComponentProps {
    label?: string;
    labelLevel?: text.Level;
    labelDirection?: direction.Direction;
    onLabelChange?: (label: string) => void;
}
export declare const axisFactory: (dir: direction.Direction) => FC<AxisProps>;
export declare const XAxis: FC<AxisProps>;
export declare const YAxis: FC<AxisProps>;
//# sourceMappingURL=Axis.d.ts.map