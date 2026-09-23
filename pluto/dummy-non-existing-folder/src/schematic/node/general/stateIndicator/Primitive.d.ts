import "./stateIndicator.css";
import { type schematic } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";
interface RenderProps extends Partial<Pick<schematic.StateIndicatorNodeConfig, "color" | "orientation" | "inlineSize" | "size">> {
    options: schematic.StateIndicatorNodeConfig["options"];
    className?: string;
    matchedOptionKey?: string | null;
    /** Colors the label while the state channel is stale. */
    staleColor?: color.Crude;
}
export declare const StateIndicator: ({ className, orientation, matchedOptionKey, options, color: colorVal, inlineSize, size, staleColor, }: RenderProps) => ReactElement;
export {};
//# sourceMappingURL=Primitive.d.ts.map