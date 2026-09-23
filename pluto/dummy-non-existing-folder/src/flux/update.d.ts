import { type query, type status, type Synnax as Client } from "@synnaxlabs/client";
import { type CrudeTimeSpan, type destructor, state, type verbs } from "@synnaxlabs/x";
import type z from "zod";
import { type InitialStatusDetailsContainer, type Result, type ResultStatus } from "./result";
/** What an update implementation receives. */
export interface UpdateParams<Input extends query.Data, Output extends query.Data = Input, StatusDetails extends z.ZodType = z.ZodNever> {
    data: Input;
    client: Client;
    setStatus: (setter: state.SetArg<ResultStatus<StatusDetails>>) => void;
    onOptimisticComplete: (data: Output) => Promise<void>;
}
/** Params for {@link createUpdate}. */
export type CreateUpdateParams<Input extends query.Data, Output extends query.Data = Input, StatusDetails extends z.ZodType = z.ZodNever> = {
    name: string;
    verbs: verbs.Verbs;
    update: (params: UpdateParams<Input, Output, StatusDetails>) => Promise<Output | false>;
} & InitialStatusDetailsContainer<StatusDetails>;
/** Return value for `useObservableUpdate`. */
export interface UseObservableUpdateReturn<Input extends query.Data> {
    /** Runs the update and reports a failure as an error status. */
    update: (data: Input, opts?: query.FetchOptions) => void;
    /** Runs the update and resolves to whether it succeeded. */
    updateAsync: (data: Input, opts?: query.FetchOptions) => Promise<boolean>;
}
export interface UseObservableUpdateParams<Input extends query.Data, Output extends query.Data = Input, StatusDetails extends z.ZodType = z.ZodNever> {
    debounce?: CrudeTimeSpan;
    onChange: state.Setter<Result<Input | undefined, StatusDetails>>;
    beforeUpdate?: (params: BeforeUpdateParams<Input>) => Promise<Input | boolean> | Input | boolean;
    afterOptimistic?: (params: AfterOptimisticParams<Output>) => Promise<void> | void;
    afterSuccess?: (params: AfterSuccessParams<Output>) => Promise<void> | void;
    afterFailure?: (params: AfterFailureParams<Input>) => Promise<void> | void;
}
export interface BeforeUpdateParams<Data extends query.Data> {
    /** Side-effect undos run in reverse order when the update fails. */
    rollbacks: destructor.Destructor[];
    client: Client;
    data: Data;
}
export interface AfterOptimisticParams<Output extends query.Data> {
    /** Side-effect undos run in reverse order when the update fails. */
    rollbacks: destructor.Destructor[];
    client: Client;
    data: Output;
}
export interface AfterSuccessParams<Output extends query.Data> {
    client: Client;
    data: Output;
}
export interface AfterFailureParams<Data extends query.Data> {
    client: Client;
    status: status.Status<typeof status.exceptionDetailsSchema, z.ZodLiteral<"error">>;
    data: Data;
}
export interface UseDirectUpdateParams<Input extends query.Data, Output extends query.Data = Input, StatusDetails extends z.ZodType = z.ZodNever> extends Omit<UseObservableUpdateParams<Input, Output, StatusDetails>, "onChange"> {
}
/** Return value for `useUpdate`: the update callbacks plus its own result state. */
export type UseDirectUpdateReturn<Input extends query.Data, StatusDetails extends z.ZodType = z.ZodNever> = Result<Input | undefined, StatusDetails> & UseObservableUpdateReturn<Input>;
export interface UseObservableUpdate<Input extends query.Data, Output extends query.Data = Input, StatusDetails extends z.ZodType = z.ZodNever> {
    (params: UseObservableUpdateParams<Input, Output, StatusDetails>): UseObservableUpdateReturn<Input>;
}
export interface UseUpdate<Input extends query.Data, Output extends query.Data = Input, StatusDetails extends z.ZodType = z.ZodNever> {
    (params?: UseDirectUpdateParams<Input, Output, StatusDetails>): UseDirectUpdateReturn<Input, StatusDetails>;
}
/** The hooks {@link createUpdate} builds for one mutation. */
export interface CreateUpdateReturn<Input extends query.Data, Output extends query.Data = Input, StatusDetails extends z.ZodType = z.ZodNever> {
    useObservableUpdate: UseObservableUpdate<Input, Output, StatusDetails>;
    useUpdate: UseUpdate<Input, Output, StatusDetails>;
}
/**
 * Builds the hooks that run one mutation against a Core, applying the change to the
 * cache before the request returns and rolling it back on failure. Call it once per
 * mutation, at module scope.
 */
export declare const createUpdate: <Input extends query.Data, Output extends query.Data = Input, StatusDetails extends z.ZodType = z.ZodNever>(createParams: CreateUpdateParams<Input, Output, StatusDetails>) => CreateUpdateReturn<Input, Output, StatusDetails>;
//# sourceMappingURL=update.d.ts.map