import { type group, type ontology } from "@synnaxlabs/client";
import { Flux } from "../flux";
export declare const RESOURCE_NAME = "group";
export interface CreateParams extends group.CreateParams {
}
export declare const useCreate: Flux.UseUpdate<CreateParams, CreateParams, import("zod").ZodNever>;
export type ListQuery = {
    parent?: ontology.ID;
    searchTerm?: string;
    offset?: number;
    limit?: number;
};
export declare const useList: Flux.UseList<ListQuery, string, group.Group>;
export interface DeleteParams {
    key: string;
}
export declare const useDelete: Flux.UseUpdate<DeleteParams, DeleteParams, import("zod").ZodNever>;
export interface RenameParams extends Pick<group.Group, "key" | "name"> {
}
export declare const useRename: Flux.UseUpdate<RenameParams, RenameParams, import("zod").ZodNever>;
//# sourceMappingURL=queries.d.ts.map