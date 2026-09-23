import { access, user } from "@synnaxlabs/client";
import { type List } from "@synnaxlabs/lyra/list";
import { z } from "zod";
import { Flux } from "../../flux";
export type RetrieveQuery = {
    key: string;
};
export declare const use: Flux.Use<RetrieveQuery, access.role.Role>;
export type ForUserQuery = {
    user: user.Key;
};
/**
 * Retrieves the roles assigned to a user. Assignment is an ontology relationship, so
 * the roles are the user's role-typed parents.
 */
export declare const useResultForUser: Flux.UseResult<ForUserQuery, access.role.Role[]>;
export type ListQuery = List.PagerParams;
export declare const useList: Flux.UseList<List.PagerParams, string, access.role.Role>;
export type DeleteParams = access.role.Key | access.role.Key[];
export declare const useDelete: Flux.UseUpdate<DeleteParams, DeleteParams, z.ZodNever>;
export interface RenameParams {
    key: access.role.Key;
    name: string;
}
export declare const useRename: Flux.UseUpdate<RenameParams, RenameParams, z.ZodNever>;
export type ChangeRoleFormQuery = {
    key: user.Key;
};
export declare const changeRoleFormSchema: z.ZodObject<{
    key: z.ZodUUID;
    role: z.ZodUUID;
}, z.core.$strip>;
export declare const useChangeRoleForm: Flux.UseForm<ChangeRoleFormQuery, z.ZodObject<{
    key: z.ZodUUID;
    role: z.ZodUUID;
}, z.core.$strip>>;
//# sourceMappingURL=queries.d.ts.map