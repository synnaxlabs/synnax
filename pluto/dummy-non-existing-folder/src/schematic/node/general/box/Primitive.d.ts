import { type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";
interface RenderProps extends Omit<schematic.BoxNodeConfig, "variant" | "label" | "scale"> {
    className?: string;
}
export declare const Box: ({ borderRadius, ...rest }: RenderProps) => ReactElement;
export {};
//# sourceMappingURL=Primitive.d.ts.map