import { type schematic } from "@synnaxlabs/client";
import { type Spec } from "./spec";
export declare const REGISTRY: {
    readonly pipe: Spec<"pipe", schematic.PipeEdgeConfig>;
    readonly electric: Spec<"electric", schematic.ElectricEdgeConfig>;
    readonly secondary: Spec<"secondary", schematic.SecondaryEdgeConfig>;
    readonly jacketed: Spec<"jacketed", schematic.JacketedEdgeConfig>;
    readonly hydraulic: Spec<"hydraulic", schematic.HydraulicEdgeConfig>;
    readonly pneumatic: Spec<"pneumatic", schematic.PneumaticEdgeConfig>;
    readonly data: Spec<"data", schematic.DataEdgeConfig>;
};
export type Variant = schematic.EdgeConfigType;
export type Config = schematic.EdgeConfig;
export declare const resolveSpec: (variant: string) => Spec<Variant, Config>;
//# sourceMappingURL=registry.d.ts.map