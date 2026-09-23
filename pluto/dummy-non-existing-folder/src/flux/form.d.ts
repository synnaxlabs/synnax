import { type query, type Synnax as Client } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { type CrudeTimeSpan, type destructor } from "@synnaxlabs/x";
import { type z } from "zod";
import { type Result } from "./result";
import { type RetrieveParams } from "./suspend";
import { type UpdateParams } from "./update";
export interface FormUpdateParams<Schema extends z.ZodType<query.Data>> extends Omit<UpdateParams<z.infer<Schema>>, "data" | "onChange" | "onOptimisticComplete">, Omit<Form.UseReturn<Schema>, "setStatus"> {
}
/** Client and query handles for a form operation. */
interface FormClientParams<Query extends query.Params> {
    client: Client;
    query: Query | null;
}
export interface CreateFormParams<Query extends query.Params, Schema extends z.ZodType<query.Data>> {
    name: string;
    schema: Schema;
    initialValues: z.infer<Schema>;
    /**
     * Fetches the record's form values. Omit for a form that never reads. The
     * hook suspends on this, so the form is built from real values rather than
     * from a placeholder the fetch overwrites.
     */
    retrieve?: (params: RetrieveParams<Query>) => Promise<z.infer<Schema>>;
    /**
     * Projects the record's form values out of the domain client's cache.
     * A hit resolves the read synchronously, with no fetch and no suspension.
     */
    getCached?: (params: RetrieveParams<Query>) => z.infer<Schema> | undefined;
    update: (params: FormUpdateParams<Schema>) => Promise<void>;
    mountListeners?: (params: FormMountListenersParams<Query, Schema>) => destructor.Destructor | destructor.Destructor[];
    /**
     * Canonicalizes the caller's query before anything reads it: `retrieve`,
     * `mountListeners`, and `getCached` all receive the one normalized,
     * identity-stable object. Merge defaults here instead of at each callback,
     * where a per-call spread would mint a fresh object and miss the client's
     * query memos. Must preserve fields it does not set.
     */
    normalizeQuery?: <Q extends Query>(query: Q) => Q;
}
export type UseFormReturn<Schema extends z.ZodType<query.Data>> = Omit<Result<z.infer<Schema>>, "data"> & {
    form: Form.UseReturn<Schema>;
    save: (opts?: query.FetchOptions) => void;
    /** Like save, but resolves true once the update has been persisted. */
    saveAsync: (opts?: query.FetchOptions) => Promise<boolean>;
};
export interface FormBeforeSaveParams<Query extends query.Params, Schema extends z.ZodType<query.Data>> extends Form.UseReturn<Schema>, FormClientParams<Query> {
}
interface FormMountListenersParams<Query extends query.Params, Schema extends z.ZodType<query.Data>> extends Form.UseReturn<Schema>, Omit<FormClientParams<Query>, "query"> {
    query: Query;
    /**
     * Drops the pending autosave, aborts one already running, and stops further ones.
     * Call when the record no longer exists: a save queued before a delete would
     * otherwise write it back.
     */
    abandon: () => void;
}
export interface AfterSaveParams<Query extends query.Params, Schema extends z.ZodType<query.Data>> extends FormBeforeSaveParams<Query, Schema> {
}
export interface BeforeValidateParams<Query extends query.Params, Schema extends z.ZodType<query.Data>> extends FormBeforeSaveParams<Query, Schema> {
}
export interface UseFormParams<Query extends query.Params, Schema extends z.ZodType<query.Data>> extends Pick<Form.UseParams<Schema>, "sync" | "onHasTouched" | "mode"> {
    initialValues?: z.infer<Schema>;
    autoSave?: boolean;
    /**
     * How long to wait after a change before autosaving. Raise it for a form with a
     * continuous input, such as a drag handle or a color picker, where one gesture
     * emits a burst of changes. Zero saves on every change.
     */
    autoSaveDebounce?: CrudeTimeSpan;
    /** The record to edit, or null for a form with nothing to read. */
    query: Query | null;
    beforeValidate?: (params: BeforeValidateParams<Query, Schema>) => boolean | void;
    beforeSave?: (params: FormBeforeSaveParams<Query, Schema>) => Promise<boolean>;
    afterSave?: (params: AfterSaveParams<Query, Schema>) => void;
}
export interface UseForm<Query extends query.Params, Schema extends z.ZodType<query.Data>> {
    (params: UseFormParams<Query, Schema>): UseFormReturn<Schema>;
}
export declare const createForm: <Query extends query.Params, Schema extends z.ZodType<query.Data>>({ name, schema, retrieve, getCached, mountListeners, update, initialValues: baseInitialValues, normalizeQuery, }: CreateFormParams<Query, Schema>) => UseForm<Query, Schema>;
export {};
//# sourceMappingURL=form.d.ts.map