import { label, type ontology, status, TimeStamp } from "@synnaxlabs/client";
import type z from "zod";
import { Flux } from "../flux";
export type ListParams = status.MultiRetrieveParams;
export declare const useList: Flux.UseList<{
    keys?: string[] | undefined;
    searchTerm?: string | undefined;
    offset?: number | undefined;
    limit?: number | undefined;
    includeLabels?: boolean | undefined;
    hasLabels?: string[] | undefined;
    variants?: string[] | undefined;
    ignoreNotFoundError?: boolean | undefined;
}, string, {
    key: status.Key;
    name: string;
    variant: "disabled" | "error" | "info" | "loading" | "success" | "warning";
    message: string;
    description: string;
    time: TimeStamp;
    labels?: label.Label[];
}>;
export declare const useDelete: Flux.UseUpdate<string | string[], string | string[], z.ZodNever>;
export interface SetParams {
    statuses: status.New | status.New[];
    parent?: ontology.ID;
}
export declare const useSet: Flux.UseUpdate<SetParams, SetParams, z.ZodNever>;
export type RetrieveQuery = status.SingleRetrieveParams;
export declare const createRetrieve: <DetailsSchema extends z.ZodType = z.ZodNever>(detailsSchema?: DetailsSchema) => Flux.CreateRetrieveReturn<{
    key: string;
    includeLabels?: boolean | undefined;
}, status.Status<DetailsSchema, z.ZodEnum<{
    disabled: "disabled";
    error: "error";
    info: "info";
    loading: "loading";
    success: "success";
    warning: "warning";
}>>>;
export type RetrieveMultipleQuery = {
    keys: status.Key[];
};
export declare const useMultiple: Flux.Use<RetrieveMultipleQuery, {
    key: status.Key;
    name: string;
    variant: "disabled" | "error" | "info" | "loading" | "success" | "warning";
    message: string;
    description: string;
    time: TimeStamp;
    labels?: label.Label[];
}[]>, useResultMultiple: Flux.UseResult<RetrieveMultipleQuery, {
    key: status.Key;
    name: string;
    variant: "disabled" | "error" | "info" | "loading" | "success" | "warning";
    message: string;
    description: string;
    time: TimeStamp;
    labels?: label.Label[];
}[]>;
export declare const use: Flux.Use<{
    key: string;
    includeLabels?: boolean | undefined;
}, {
    key: status.Key;
    name: string;
    variant: "disabled" | "error" | "info" | "loading" | "success" | "warning";
    message: string;
    description: string;
    time: TimeStamp;
    labels?: label.Label[];
}>, useResult: Flux.UseResult<{
    key: string;
    includeLabels?: boolean | undefined;
}, {
    key: status.Key;
    name: string;
    variant: "disabled" | "error" | "info" | "loading" | "success" | "warning";
    message: string;
    description: string;
    time: TimeStamp;
    labels?: label.Label[];
}>;
export declare const useSetSynchronizer: (onSet: (status: status.Status) => void) => void;
export declare const useDeleteSynchronizer: (onDelete: (key: status.Key) => void) => void;
export declare const formSchema: z.ZodObject<{
    key: z.ZodDefault<typeof status.keyZ>;
    name: z.ZodDefault<z.ZodString>;
    variant: z.ZodEnum<{
        disabled: "disabled";
        error: "error";
        info: "info";
        loading: "loading";
        success: "success";
        warning: "warning";
    }>;
    message: z.ZodString;
    description: z.ZodDefault<z.ZodString>;
    time: z.ZodDefault<typeof import("@synnaxlabs/x").timeStampZ>;
    details: z.ZodOptional<z.ZodUnknown>;
    labels: z.ZodOptional<z.ZodArray<z.ZodUUID>>;
}, z.core.$strip>;
export declare const useForm: Flux.UseForm<{
    key: string;
    includeLabels?: boolean | undefined;
}, z.ZodObject<{
    key: z.ZodDefault<typeof status.keyZ>;
    name: z.ZodDefault<z.ZodString>;
    variant: z.ZodEnum<{
        disabled: "disabled";
        error: "error";
        info: "info";
        loading: "loading";
        success: "success";
        warning: "warning";
    }>;
    message: z.ZodString;
    description: z.ZodDefault<z.ZodString>;
    time: z.ZodDefault<typeof import("@synnaxlabs/x").timeStampZ>;
    details: z.ZodOptional<z.ZodUnknown>;
    labels: z.ZodOptional<z.ZodArray<z.ZodUUID>>;
}, z.core.$strip>>;
export interface RenameParams extends Pick<status.Status, "key" | "name"> {
}
export declare const useRename: Flux.UseUpdate<RenameParams, RenameParams, z.ZodNever>;
//# sourceMappingURL=queries.d.ts.map