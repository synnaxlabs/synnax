import { ontology } from "@synnaxlabs/client";
import { type List } from "@synnaxlabs/lyra/list";
import { Flux } from "../flux";
export declare const useResourceSetSynchronizer: (onSet: (resource: ontology.Resource) => void) => void;
export declare const useResourceDeleteSynchronizer: (onDelete: (id: ontology.ID) => void) => void;
export declare const useRelationshipSetSynchronizer: (onSet: (relationship: ontology.Relationship) => void) => void;
export declare const useRelationshipDeleteSynchronizer: (onDelete: (relationship: ontology.Relationship) => void) => void;
type DependentQuery = List.PagerParams & {
    id?: ontology.ID;
};
export declare const createDependentsListHook: (direction: ontology.RelationshipDirection, name: string) => Flux.UseList<DependentQuery, string, ontology.Resource<import("@synnaxlabs/x/dist/src/record/record.js").Unknown>>;
export declare const useListChildren: Flux.UseList<DependentQuery, string, ontology.Resource<import("@synnaxlabs/x/dist/src/record/record.js").Unknown>>;
export type ListQuery = ontology.RetrieveRequest;
export declare const useResourceList: Flux.UseList<ontology.RetrieveRequest, string, ontology.Resource<import("@synnaxlabs/x/dist/src/record/record.js").Unknown>>;
export interface MoveChildrenParams {
    source: ontology.ID;
    destination: ontology.ID;
    ids: ontology.ID[];
}
export declare const useMoveChildren: Flux.UseUpdate<MoveChildrenParams, MoveChildrenParams, import("zod").ZodNever>;
export type RetrieveChildrenQuery = {
    [K in keyof ontology.RetrieveOptions]: ontology.RetrieveOptions[K];
} & {
    id: ontology.ID;
};
export declare const useChildren: Flux.Use<RetrieveChildrenQuery, ontology.Resource<import("@synnaxlabs/x/dist/src/record/record.js").Unknown>[]>;
type RetrieveResourceQuery = {
    ids: ontology.ID[];
};
export declare const useResource: Flux.Use<RetrieveResourceQuery, ontology.Resource<import("@synnaxlabs/x/dist/src/record/record.js").Unknown>[]>;
export {};
//# sourceMappingURL=queries.d.ts.map