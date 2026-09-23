import { type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";
interface RenderProps extends Partial<Pick<schematic.LightNodeConfig, "color" | "orientation" | "scale">> {
    className?: string;
    enabled?: boolean;
}
export declare const WIDTH_PER_SCALE: number;
export declare const Light: ({ className, color, orientation, enabled, scale, }: RenderProps) => ReactElement;
export {};
//# sourceMappingURL=Primitive.d.ts.map