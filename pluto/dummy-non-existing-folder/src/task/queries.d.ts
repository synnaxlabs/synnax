import { type ontology, rack, task } from "@synnaxlabs/client";
import { type optional } from "@synnaxlabs/x";
import { z } from "zod";
import { Flux } from "../flux";
export declare const RESOURCE_NAME = "task";
export type RetrieveQuery = task.RetrieveSingleParams;
export declare const createRetrieve: <S extends task.Schemas = task.Schemas>(schemas?: S) => Flux.CreateRetrieveReturn<{
    key: string;
    includeStatus?: boolean | undefined;
} | {
    name: string;
    includeStatus?: boolean | undefined;
} | {
    type: string;
    rack?: number | undefined;
}, task.Task<S>>;
export declare const use: Flux.Use<{
    key: string;
    includeStatus?: boolean | undefined;
} | {
    name: string;
    includeStatus?: boolean | undefined;
} | {
    type: string;
    rack?: number | undefined;
}, task.Task<task.PayloadSchemas<z.ZodType<string, unknown, z.core.$ZodTypeInternals<string, unknown>>, z.ZodType<import("@synnaxlabs/x/dist/src/record/record.js").Unknown, unknown, z.core.$ZodTypeInternals<import("@synnaxlabs/x/dist/src/record/record.js").Unknown, unknown>>, z.ZodType<unknown, unknown, z.core.$ZodTypeInternals<unknown, unknown>>>>>, useEnsure: Flux.UseEnsure<{
    key: string;
    includeStatus?: boolean | undefined;
} | {
    name: string;
    includeStatus?: boolean | undefined;
} | {
    type: string;
    rack?: number | undefined;
}>, useResult: Flux.UseResult<{
    key: string;
    includeStatus?: boolean | undefined;
} | {
    name: string;
    includeStatus?: boolean | undefined;
} | {
    type: string;
    rack?: number | undefined;
}, task.Task<task.PayloadSchemas<z.ZodType<string, unknown, z.core.$ZodTypeInternals<string, unknown>>, z.ZodType<import("@synnaxlabs/x/dist/src/record/record.js").Unknown, unknown, z.core.$ZodTypeInternals<import("@synnaxlabs/x/dist/src/record/record.js").Unknown, unknown>>, z.ZodType<unknown, unknown, z.core.$ZodTypeInternals<unknown, unknown>>>>>, useTombstone: Flux.UseTombstone<{
    key: string;
    includeStatus?: boolean | undefined;
} | {
    name: string;
    includeStatus?: boolean | undefined;
} | {
    type: string;
    rack?: number | undefined;
}>, createSelector: Flux.CreateSelector<{
    key: string;
    includeStatus?: boolean | undefined;
} | {
    name: string;
    includeStatus?: boolean | undefined;
} | {
    type: string;
    rack?: number | undefined;
}, task.Task<task.PayloadSchemas<z.ZodType<string, unknown, z.core.$ZodTypeInternals<string, unknown>>, z.ZodType<import("@synnaxlabs/x/dist/src/record/record.js").Unknown, unknown, z.core.$ZodTypeInternals<import("@synnaxlabs/x/dist/src/record/record.js").Unknown, unknown>>, z.ZodType<unknown, unknown, z.core.$ZodTypeInternals<unknown, unknown>>>>>;
export interface KeyParams {
    key: task.Key;
}
export declare const useName: Flux.UseSelect<{
    key: string;
    includeStatus?: boolean | undefined;
} | {
    name: string;
    includeStatus?: boolean | undefined;
} | {
    type: string;
    rack?: number | undefined;
}, string>;
export type ListQuery = task.RetrieveMultipleParams;
export declare const useList: Flux.UseList<{
    rack?: number | undefined;
    keys?: string[] | undefined;
    names?: string[] | undefined;
    types?: string[] | undefined;
    includeStatus?: boolean | undefined;
    internal?: boolean | undefined;
    snapshot?: boolean | undefined;
    searchTerm?: string | undefined;
    offset?: number | undefined;
    limit?: number | undefined;
}, string, task.Task<task.PayloadSchemas<z.ZodType<string, unknown, z.core.$ZodTypeInternals<string, unknown>>, z.ZodType<import("@synnaxlabs/x/dist/src/record/record.js").Unknown, unknown, z.core.$ZodTypeInternals<import("@synnaxlabs/x/dist/src/record/record.js").Unknown, unknown>>, z.ZodType<unknown, unknown, z.core.$ZodTypeInternals<unknown, unknown>>>>>;
export interface FormSchema<S extends task.Schemas = task.Schemas> extends z.ZodType<{
    key?: task.Key;
    name: string;
    rack: rack.Key;
    type: z.infer<S["type"]>;
    snapshot: boolean;
    config: z.infer<S["config"]>;
    configHash: string;
    status?: task.Status<S["statusData"]>;
}> {
}
export interface CreateFormParams<S extends task.Schemas = task.Schemas> {
    schemas: S;
    initialValues: InitialValues<S>;
}
export interface InitialValues<S extends task.Schemas = task.Schemas> extends optional.Optional<task.Payload<S>, "key" | "rack" | "internal" | "snapshot" | "configHash"> {
}
export type FormQuery = {
    key: task.Key;
};
export declare const toFormValues: <S extends task.Schemas = task.Schemas>(t: InitialValues<S>) => z.infer<FormSchema<S>>;
export declare const createForm: <S extends task.Schemas = task.Schemas>({ schemas, initialValues, }: CreateFormParams<S>) => Flux.UseForm<FormQuery, FormSchema<S>>;
export type DeleteParams = task.Key | task.Key[];
export declare const useDelete: Flux.UseUpdate<DeleteParams, DeleteParams, z.ZodNever>;
export interface SnapshotPair extends Pick<task.Payload, "key" | "name"> {
}
export interface SnapshotParams {
    tasks: SnapshotPair | SnapshotPair[];
    parentID: ontology.ID;
}
export declare const useCreateSnapshot: Flux.UseUpdate<SnapshotParams, SnapshotParams, z.ZodNever>;
export interface UseRenameParams extends Pick<task.Payload, "key" | "name"> {
}
export declare const useRename: Flux.UseUpdate<UseRenameParams, UseRenameParams, z.ZodNever>;
export type CommandParams = task.NewCommand | task.NewCommand[];
export declare const shouldExecuteCommand: <StatusData extends z.ZodType = z.ZodNever>(status: task.Status<StatusData>, command: string) => boolean;
export declare const useCommand: Flux.UseUpdate<CommandParams, CommandParams, z.ZodNever>;
//# sourceMappingURL=queries.d.ts.map