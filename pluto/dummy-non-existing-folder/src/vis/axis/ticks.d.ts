import { type scale } from "@synnaxlabs/x";
import { z } from "zod";
export interface Tick {
    position: number;
    label: string;
}
export interface TickFactory {
    create: (params: TickFactoryRenderParams) => Tick[];
}
export declare const tickType: z.ZodEnum<{
    linear: "linear";
    time: "time";
}>;
export type TickType = z.infer<typeof tickType>;
export declare const tickFactoryProps: z.ZodObject<{
    tickSpacing: z.ZodDefault<z.ZodNumber>;
    type: z.ZodDefault<z.ZodEnum<{
        linear: "linear";
        time: "time";
    }>>;
}, z.core.$strip>;
export type TickFactoryProps = z.input<typeof tickFactoryProps>;
export interface TickFactoryRenderParams {
    /** Scale takes a value in decimal space and returns the corresponding data value. */
    decimalToDataScale: scale.Scale;
    /** Size is the length of the axis in pixels. */
    size: number;
}
export declare const newTickFactory: (props: TickFactoryProps) => TickFactory;
//# sourceMappingURL=ticks.d.ts.map