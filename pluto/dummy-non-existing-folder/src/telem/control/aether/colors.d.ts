import { control, type Synnax } from "@synnaxlabs/client";
import { color, type destructor } from "@synnaxlabs/x";
import { z } from "zod";
import { aether } from "../../../aether/aether";
export declare const colorsStateZ: z.ZodObject<{}, z.core.$strip>;
interface InternalState {
    palette: color.Color[];
    defaultColor: color.Color;
    client: Synnax | null;
}
/**
 * A control state with the color assigned to the subject holding it, which is how
 * control state reaches the main thread.
 */
export declare const coloredStateZ: z.ZodObject<{
    subject: z.ZodObject<{
        key: z.ZodString;
        name: z.ZodString;
        group: z.ZodDefault<z.ZodUInt32>;
    }, z.core.$strip>;
    resource: z.ZodNumber;
    authority: z.ZodInt;
    subjectColor: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | [number, number, number] | [number, number, number, number]>>;
}, z.core.$strip>;
export interface ColoredState extends control.State {
    subjectColor: color.Color;
}
/**
 * Colors assigns each control subject a color, so that a reader can tell subjects
 * apart at a glance. Assignment is sequential over the visualization palette, which
 * keeps colors unique for as many subjects as the palette holds.
 */
export declare class Colors extends aether.Composite<typeof colorsStateZ, InternalState> {
    static readonly TYPE = "Colors";
    static readonly stateZ: z.ZodObject<{}, z.core.$strip>;
    schema: z.ZodObject<{}, z.core.$strip>;
    /** The color assigned to each control subject, keyed by subject key. */
    private readonly assigned;
    /** User-specified color overrides from the legend UI. */
    private readonly overrides;
    private readonly obs;
    private disconnect?;
    /**
     * Grabs the color assignment from the current aether context.
     *
     * @param ctx - The component's current aether context.
     * @throws {Error} if it is not in the context.
     */
    static use(ctx: aether.Context): Colors;
    afterUpdate(ctx: aether.Context): void;
    afterDelete(): void;
    /** Subscribes to changes in the assignment. The handler takes no arguments: it
     * re-reads the colors it needs through {@link get}. */
    onChange(handler: () => void): destructor.Destructor;
    /** The color of the given subject, or the default color when it has none. */
    get(subject: string): color.Color;
    setOverrides(overrides: Record<string, color.Color>): void;
    /**
     * Assigns a color to every subject holding control, and releases the colors of
     * subjects that hold none. Colors are assigned over the subjects the client has
     * cached, so an unrelated subject elsewhere in the cluster never shifts them.
     */
    private assign;
}
export {};
//# sourceMappingURL=colors.d.ts.map