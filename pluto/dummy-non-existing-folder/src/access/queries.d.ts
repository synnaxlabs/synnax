import { type access, type ontology, type Synnax } from "@synnaxlabs/client";
import { Flux } from "../flux";
export type PermissionsQuery = {
    subject?: ontology.ID;
    objects: ontology.ID | ontology.ID[];
    action: access.Action;
};
export interface IsGrantedParams {
    client: Synnax | null;
    query: PermissionsQuery;
}
export declare const isGranted: ({ client, query: { subject, objects, action }, }: IsGrantedParams) => boolean;
export interface IsGrantedExtensionParams extends Omit<IsGrantedParams, "query"> {
}
export declare const useGranted: (query: PermissionsQuery) => boolean;
export declare const useRetrieveGranted: (id: ontology.ID | ontology.ID[]) => boolean;
export declare const useUpdateGranted: (id: ontology.ID | ontology.ID[]) => boolean;
export declare const useDeleteGranted: (id: ontology.ID | ontology.ID[]) => boolean;
export declare const useCreateGranted: (id: ontology.ID | ontology.ID[]) => boolean;
export interface GrantedParams extends Omit<IsGrantedParams, "query"> {
    id: ontology.ID | ontology.ID[];
}
export declare const viewGranted: ({ id, ...rest }: GrantedParams) => boolean;
export declare const updateGranted: ({ id, ...rest }: GrantedParams) => boolean;
export declare const deleteGranted: ({ id, ...rest }: GrantedParams) => boolean;
export declare const createGranted: ({ id, ...rest }: GrantedParams) => boolean;
export type LoadPermissionsQuery = {
    subject?: ontology.ID;
};
/**
 * useEnsurePermissions suspends until the subject's policies are cached, so a surface
 * mounted below it never reads an empty policy set as a denial. A failed read throws
 * to the surrounding boundary; call useInvalidatePermissions before resetting it, or
 * the settled failure throws again on the next render.
 */
export declare const useEnsurePermissions: Flux.UseEnsure<LoadPermissionsQuery>, useInvalidatePermissions: Flux.UseInvalidate<LoadPermissionsQuery>;
//# sourceMappingURL=queries.d.ts.map