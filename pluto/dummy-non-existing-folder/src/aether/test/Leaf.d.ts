import { z } from "zod";
import { aether } from "../aether";
export declare const leafStateZ: z.ZodRecord<z.ZodString, z.ZodUnknown>;
export type LeafState = z.infer<typeof leafStateZ>;
/** Captures a single `afterUpdate` invocation on a {@link Leaf}, including the
 * state at that point and the state immediately preceding it. */
export interface UpdateCall {
    state: LeafState;
    prevState: LeafState;
}
/** Bundled aether `Leaf` stub for tests. Registered in a test registry under any type
 * name; records every lifecycle invocation for later assertion. The schema accepts any
 * record-shaped state. */
export declare class Leaf extends aether.Leaf<typeof leafStateZ> {
    static readonly TYPE: string;
    static readonly stateZ: z.ZodRecord<z.ZodString, z.ZodUnknown>;
    schema: z.ZodRecord<z.ZodString, z.ZodUnknown>;
    /** Every call to {@link afterUpdate}, in order. */
    readonly updateCalls: UpdateCall[];
    /** Number of `afterDelete` invocations. */
    deleteCallCount: number;
    afterUpdate(): void;
    afterDelete(): void;
}
//# sourceMappingURL=Leaf.d.ts.map