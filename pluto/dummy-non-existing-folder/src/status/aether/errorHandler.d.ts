import { status } from "@synnaxlabs/client";
import { type errors } from "@synnaxlabs/x";
import type z from "zod";
export interface Adder {
    <Details extends z.ZodType = z.ZodNever>(spec: status.Crude<Details>): void;
}
export interface ErrorHandler {
    /** Reports the given error, or runs the given function and reports a rejection. */
    (funcOrExc: unknown, message?: string, skip?: errors.Matchable | errors.Matchable[]): void;
}
export interface AsyncErrorHandler {
    /** Reports the given error, or runs the given function and reports a rejection. */
    (funcOrExc: unknown, message?: string, skip?: errors.Matchable | errors.Matchable[]): Promise<void>;
}
export declare const createErrorHandler: (add: Adder) => ErrorHandler;
export declare const createAsyncErrorHandler: (add: Adder) => AsyncErrorHandler;
//# sourceMappingURL=errorHandler.d.ts.map