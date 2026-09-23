import "./stringDisplay.css";
import { type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";
interface RenderProps extends Partial<Pick<schematic.StringDisplayNodeConfig, "color" | "textColor" | "stalenessColor" | "orientation" | "level" | "inlineSize">> {
    className?: string;
    value?: string;
    stale?: boolean;
}
export declare const StringDisplay: ({ className, color: colorVal, textColor, stalenessColor, level, orientation, inlineSize, value, stale, }: RenderProps) => ReactElement;
export {};
//# sourceMappingURL=Primitive.d.ts.map