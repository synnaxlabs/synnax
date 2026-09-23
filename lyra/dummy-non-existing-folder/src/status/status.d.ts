import { type optional, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";
export declare const VARIANTS: readonly ["success", "info", "warning", "error", "loading", "disabled"];
export declare const variantZ: z.ZodEnum<{
    disabled: "disabled";
    error: "error";
    info: "info";
    loading: "loading";
    success: "success";
    warning: "warning";
}>;
export type Variant = z.infer<typeof variantZ>;
interface Base<V extends Variant> {
    key: string;
    name: string;
    variant: V;
    message: string;
    description: string;
    time: TimeStamp;
}
/** A status carrying optional structured details. */
export type Status<Details extends z.ZodType = z.ZodNever, V extends Variant = Variant> = Base<V> & ([Details] extends [z.ZodNever] ? {} : {
    details: z.output<Details>;
});
/** A status before it is stamped with a key, a time, a name, and a description. */
export type Crude<Details extends z.ZodType = z.ZodNever, V extends Variant = Variant> = optional.Optional<Base<V>, "key" | "time" | "name" | "description"> & ([Details] extends [z.ZodNever] ? {} : {
    details: z.output<Details>;
});
/** Stamps a {@link Crude} status with a fresh key and the current time. */
export declare const create: <Details extends z.ZodType = z.ZodNever, V extends Variant = Variant>(spec: Crude<Details, V>) => Status<Details, V>;
/** Details a status built by {@link fromException} carries. */
export declare const exceptionDetailsZ: z.ZodObject<{
    stack: z.ZodString;
    error: z.ZodCustom<Error, Error>;
}, z.core.$strip>;
/**
 * Turns any thrown value into an error status. The message argument becomes the
 * status message and pushes the error's own message into the description, ahead of
 * its flattened cause chain; the error and its stack ride on the details.
 */
export declare const fromException: (exc: unknown, message?: string) => Status<typeof exceptionDetailsZ, "error">;
/** @returns the variant unless it is one of the removed ones. */
export declare const removeVariants: (variant?: Variant, remove?: Variant | Variant[]) => Variant | undefined;
/** Renders a status as readable text, pretty-printing a JSON description. */
export declare const toString: <Details extends z.ZodType = z.ZodNever>(stat: Status<Details>) => string;
export {};
//# sourceMappingURL=status.d.ts.map