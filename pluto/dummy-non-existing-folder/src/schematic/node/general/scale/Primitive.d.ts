import { type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";
interface RenderProps extends Pick<schematic.ScaleNodeConfig, "indicator"> {
    className?: string;
}
export declare const Scale: ({ indicator: { color: c, axisColor, fillHidden, caretHidden }, className, }: RenderProps) => ReactElement;
export {};
//# sourceMappingURL=Primitive.d.ts.map