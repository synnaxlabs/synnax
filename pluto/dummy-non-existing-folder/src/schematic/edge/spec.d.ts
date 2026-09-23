import { type schematic } from "@synnaxlabs/client";
import { type FC } from "react";
import { type z } from "zod";
import { type FormProps } from "../node/spec";
import { type Diagram } from "../../vis/diagram";
export interface EdgeProps extends Diagram.EdgeProps {
    onChange: (p: Partial<schematic.EdgeConfig>) => void;
    config: schematic.EdgeConfig;
}
export type Edge = FC<EdgeProps>;
export interface Spec<Variant extends string = string, P extends {
    variant: Variant;
} = {
    variant: Variant;
}> {
    key: Variant;
    name: string;
    configZ: z.ZodType<P>;
    Form: FC<FormProps>;
    Edge: Edge;
    defaultConfig: () => P;
}
//# sourceMappingURL=spec.d.ts.map