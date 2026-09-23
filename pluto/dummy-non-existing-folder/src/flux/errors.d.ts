import { errors } from "@synnaxlabs/x";
declare const FluxError_base: errors.TypedClass;
/** Base for every error Flux raises. Match on it to catch them all. */
export declare class FluxError extends FluxError_base {
}
declare const Base: errors.TypedClass;
/** The remains of a deleted record, read in place of the record itself. */
export interface Tombstone {
    /** The deleted record's name, absent when the corpse carried none. */
    name?: string;
}
/** Builds the tombstone a caller renders for the given corpse. */
export declare const tombstoneOf: (corpse: unknown) => Tombstone;
/**
 * Thrown to the error boundary when a query's cached answer is a deletion. The
 * corpse itself stays in the client's cache, typed; only the deleted record's
 * name travels with the error, for display.
 */
export declare class DeletedError extends Base {
    /** The deleted record's name, absent when the corpse carried none. */
    readonly corpseName?: string;
    constructor(message: string, corpse: unknown);
    static readonly matches: (e: unknown) => e is DeletedError;
}
export {};
//# sourceMappingURL=errors.d.ts.map