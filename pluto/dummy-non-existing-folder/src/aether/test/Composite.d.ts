import { z } from "zod";
import { aether } from "../aether";
import { type UpdateCall } from "./Leaf";
export declare const compositeStateZ: z.ZodRecord<z.ZodString, z.ZodUnknown>;
export type CompositeState = z.infer<typeof compositeStateZ>;
/** Bundled aether `Composite` stub for tests. Same shape as {@link Leaf} but
 * accepts arbitrary descendants — drop it into a registry to stand in for a real
 * Composite parent while keeping its children under the harness's normal child
 * registry. */
export declare class Composite extends aether.Composite<typeof compositeStateZ> {
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
//# sourceMappingURL=Composite.d.ts.map