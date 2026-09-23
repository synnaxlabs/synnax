import { type z } from "zod";
import { type ContextValue, type Mode } from "./Context";
export interface OnChangeParams<Z extends z.ZodType> {
    /** The values in the form AFTER the change. */
    values: z.infer<Z>;
    /** The path that was changed. */
    path: string;
    /** The previous value at the path. */
    prev: unknown;
    /** Whether validation succeeded. */
    valid: boolean;
}
/** Params for {@link use}. */
export interface UseParams<Z extends z.ZodType> {
    /** The starting values. They are copied, not held. */
    values: z.infer<Z>;
    mode?: Mode;
    /** Whether to reset the form whenever `values` changes identity. */
    sync?: boolean;
    onChange?: (props: OnChangeParams<Z>) => void;
    /** Called when the form goes from untouched to touched, and back. */
    onHasTouched?: (value: boolean) => void;
    /** Zod schema the values validate against on every write. */
    schema?: Z;
    scope?: string;
}
export interface UseReturn<Z extends z.ZodType> extends ContextValue<Z> {
}
/**
 * Creates a form over the given values, validated by the given Zod schema. The returned
 * value is stable, so a field hook re-renders only when its own path changes.
 *
 * @example
 * const form = Form.use({ values: initial, schema: rangeZ, onChange: save });
 * return <Form.Form {...form}><Form.TextField path="name" /></Form.Form>;
 */
export declare const use: <Z extends z.ZodType>({ values: initialValues, sync, schema, mode, onChange, onHasTouched, }: UseParams<Z>) => UseReturn<Z>;
//# sourceMappingURL=use.d.ts.map