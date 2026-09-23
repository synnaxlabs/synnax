import { ontology, project, type Synnax } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";
import type z from "zod";
import { Flux } from "../flux";
export type RetrieveQuery = {
    key: project.Key;
};
export declare const use: Flux.Use<RetrieveQuery, project.Project>;
export type ListParams = Pick<project.RetrieveRequest, "keys" | "offset" | "limit" | "searchTerm">;
export declare const useList: Flux.UseList<ListParams, string, project.Project>;
export type DeleteParams = project.Key | project.Key[];
export declare const useDelete: Flux.UseUpdate<DeleteParams, DeleteParams, z.ZodNever>;
export interface RenameParams {
    key: project.Key;
    name: string;
}
export declare const useRename: Flux.UseUpdate<RenameParams, RenameParams, z.ZodNever>;
export type RetrieveGroupQuery = Record<string, never>;
export declare const useGroupID: Flux.Use<RetrieveGroupQuery, {
    type: "arc" | "builtin" | "channel" | "device" | "framer" | "group" | "label" | "lineplot" | "log" | "node" | "panel" | "policy" | "project" | "rack" | "range" | "range-alias" | "role" | "schematic" | "schematic_symbol" | "status" | "table" | "task" | "user" | "view";
    key: string;
} | {
    type: "arc" | "builtin" | "channel" | "device" | "framer" | "group" | "label" | "lineplot" | "log" | "node" | "panel" | "policy" | "project" | "rack" | "range" | "range-alias" | "role" | "schematic" | "schematic_symbol" | "status" | "table" | "task" | "user" | "view";
    key: string;
} | undefined>;
export declare const formSchema: z.ZodObject<{
    key: z.ZodOptional<z.ZodDefault<z.ZodUUID>>;
    name: z.ZodString;
    layout: z.ZodDefault<z.ZodRecord<z.ZodUnion<readonly [z.ZodString, z.ZodNumber]>, z.ZodUnknown>>;
}, z.core.$strip>;
export declare const useForm: Flux.UseForm<RetrieveQuery, z.ZodObject<{
    key: z.ZodOptional<z.ZodDefault<z.ZodUUID>>;
    name: z.ZodString;
    layout: z.ZodDefault<z.ZodRecord<z.ZodUnion<readonly [z.ZodString, z.ZodNumber]>, z.ZodUnknown>>;
}, z.core.$strip>>;
export interface SaveLayoutParams extends project.SetLayoutParams {
}
export declare const useSaveLayout: Flux.UseUpdate<SaveLayoutParams, SaveLayoutParams, z.ZodNever>;
export type RetrieveChildrenQuery = {
    resourceID?: ontology.ID;
    types: ontology.ResourceType[];
};
export interface Child extends record.KeyedNamed {
    type: ontology.ResourceType;
}
/**
 * Retrieves the sibling resources sharing the queried resource's project, excluding the
 * resource itself. Returns an empty list when the resource has no project ancestor.
 */
export declare const retrieveChildren: (client: Synnax, { resourceID, types }: RetrieveChildrenQuery) => Promise<Child[]>;
//# sourceMappingURL=queries.d.ts.map