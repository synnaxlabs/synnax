import { schematic } from "@synnaxlabs/client";
import { type z } from "zod";
export type Config<V extends schematic.EdgeConfigType = schematic.EdgeConfigType> = Extract<schematic.EdgeConfig, {
    variant: V;
}>;
export declare const createConfigZ: <V extends schematic.EdgeConfigType>(variant: V) => z.ZodType<Config<V>>;
export declare const createDefaultConfig: <V extends schematic.EdgeConfigType>(variant: V) => Config<V>;
//# sourceMappingURL=config.d.ts.map