import { observe } from "@synnaxlabs/x";
import { z } from "zod";
import { type Status } from "../status";
export interface FieldState<V = unknown> {
    value: V;
    status: Status.Crude;
    touched: boolean;
    required: boolean;
}
export interface RequiredGetOptions {
    optional?: false;
    defaultValue?: undefined;
}
export interface DefaultGetOptions<V> {
    optional?: boolean;
    /** The value returned while the path holds none. Only a change writes it. */
    defaultValue: V;
}
export interface OptionalGetOptions {
    optional: true;
    defaultValue?: undefined;
}
export interface ExtensionGetOptions<V> {
    optional?: boolean;
    defaultValue?: V;
}
export type GetOptions<V> = RequiredGetOptions | OptionalGetOptions | DefaultGetOptions<V> | ExtensionGetOptions<V>;
interface SetValueOptions {
    markTouched?: boolean;
}
export declare class State<Z extends z.ZodType> extends observe.Observer<void> {
    private readonly schema?;
    values: z.infer<Z>;
    initialValues: z.infer<Z>;
    private readonly statuses;
    private readonly touched;
    private readonly cachedRefs;
    constructor(values: z.infer<Z>, schema?: Z);
    setValue(path: string, value: unknown, options?: SetValueOptions): void;
    private checkTouched;
    private clearOrphanedTouchedPaths;
    setStatus(path: string, status: Status.Crude): void;
    clearStatus(path?: string): void;
    reset(initialValues?: z.infer<Z>): void;
    setCurrentStateAsInitialValues(): void;
    setTouched(path: string): void;
    clearTouched(path?: string): void;
    remove(path: string): void;
    validate(validateUntouched?: boolean, path?: string): boolean;
    validateAsync(validateUntouched?: boolean, path?: string): Promise<boolean>;
    private errorsToStatuses;
    private processValidation;
    get hasBeenTouched(): boolean;
    getStatuses(): Status.Crude[];
    getState<V>(path: string, opts?: RequiredGetOptions | DefaultGetOptions<V>): FieldState<V>;
    getState<V>(path: string, opts?: OptionalGetOptions | ExtensionGetOptions<V>): FieldState<V> | null;
    private updateCachedRefs;
}
export {};
//# sourceMappingURL=state.d.ts.map