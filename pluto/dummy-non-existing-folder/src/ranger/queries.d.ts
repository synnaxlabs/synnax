import { label, type ontology, ranger } from "@synnaxlabs/client";
import { type List } from "@synnaxlabs/lyra/list";
import { z } from "zod";
import { Flux } from "../flux";
import { type ListQuery } from "./aether/queries";
export type RetrieveQuery = Pick<ranger.RetrieveRequest, "includeLabels" | "includeParent"> & {
    key: ranger.Key;
};
export declare const useSetSynchronizer: (onSet: (range: ranger.Payload) => void) => void;
export declare const useDeleteSynchronizer: (onDelete: (key: ranger.Key) => void) => void;
export type ListChildrenQuery = List.PagerParams & {
    key?: ranger.Key;
};
export declare const useListChildren: Flux.UseList<ListChildrenQuery, string, ranger.Range>;
export type RetrieveParentQuery = {
    id: ontology.ID;
};
export declare const useParent: Flux.Use<RetrieveParentQuery, ranger.Range | null>, useResultParent: Flux.UseResult<RetrieveParentQuery, ranger.Range | null>;
export declare const use: Flux.Use<RetrieveQuery, ranger.Range>, useEnsure: Flux.UseEnsure<RetrieveQuery>, useTombstone: Flux.UseTombstone<RetrieveQuery>, createSelector: Flux.CreateSelector<RetrieveQuery, ranger.Range>, useResult: Flux.UseResult<RetrieveQuery, ranger.Range>;
export type RetrieveMultipleQuery = {
    keys: ranger.Key[];
};
export declare const useMultiple: Flux.Use<RetrieveMultipleQuery, ranger.Range[]>, useResultMultiple: Flux.UseResult<RetrieveMultipleQuery, ranger.Range[]>;
export declare const formSchema: z.ZodObject<{
    key: z.ZodOptional<z.ZodUUID>;
    name: z.ZodString;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | [number, number, number] | [number, number, number, number]>>>;
    labels: z.ZodArray<z.ZodUUID>;
    parent: z.ZodOptional<z.ZodString>;
    timeRange: z.ZodObject<{
        start: z.ZodNumber;
        end: z.ZodNumber;
    }, z.core.$strip>;
}, z.core.$strip>;
export declare const toFormValues: (range: ranger.Range) => z.infer<typeof formSchema>;
export type FormQuery = RetrieveQuery;
export declare const useForm: Flux.UseForm<RetrieveQuery, z.ZodObject<{
    key: z.ZodOptional<z.ZodUUID>;
    name: z.ZodString;
    color: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodNumber;
    }, z.core.$strip>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>, z.ZodObject<{
        rgba255: z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>], null>;
    }, z.core.$strip>, z.ZodObject<{
        r: z.ZodInt;
        g: z.ZodInt;
        b: z.ZodInt;
        a: z.ZodPipe<z.ZodNumber, z.ZodTransform<number, number>>;
    }, z.core.$strip>]>, z.ZodTransform<[number, number, number, number], string | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | [number, number, number] | [number, number, number, number]>>>;
    labels: z.ZodArray<z.ZodUUID>;
    parent: z.ZodOptional<z.ZodString>;
    timeRange: z.ZodObject<{
        start: z.ZodNumber;
        end: z.ZodNumber;
    }, z.core.$strip>;
}, z.core.$strip>>;
export declare const useLabels: (key: ranger.Key | null) => label.Label[] | undefined;
export { type ListQuery } from "./aether/queries";
export declare const useList: Flux.UseList<ListQuery, string, ranger.Range>;
export declare const metaDataFormSchema: z.ZodObject<{
    pairs: z.ZodArray<z.ZodObject<{
        key: z.ZodString;
        value: z.ZodString;
    }, z.core.$strip>>;
}, z.core.$strip>;
export type ListMetaDataQuery = {
    rangeKey: ranger.Key;
};
export declare const useListMetaData: Flux.UseList<ListMetaDataQuery, string, ranger.kv.Pair>;
export declare const kvPairFormSchema: z.ZodObject<{
    range: z.ZodUUID;
    key: z.ZodString;
    value: z.ZodString;
}, z.core.$strip>;
export type KVFormQuery = ListMetaDataQuery & {
    key: string;
};
export declare const useKVPairForm: Flux.UseForm<KVFormQuery, z.ZodObject<{
    range: z.ZodUUID;
    key: z.ZodString;
    value: z.ZodString;
}, z.core.$strip>>;
export interface DeleteKVParams extends ListMetaDataQuery {
    key: string;
}
export declare const useDeleteKV: Flux.UseUpdate<DeleteKVParams, DeleteKVParams, z.ZodNever>;
export interface SetKVParams extends ListMetaDataQuery, ranger.kv.Pair {
}
export declare const useUpdateKV: Flux.UseUpdate<SetKVParams, SetKVParams, z.ZodNever>;
export declare const useCreate: Flux.UseUpdate<ranger.New, ranger.Payload, z.ZodNever>;
export type DeleteParams = ranger.Key | ranger.Key[];
export declare const useDelete: Flux.UseUpdate<DeleteParams, DeleteParams, z.ZodNever>;
export interface RenameParams extends Pick<ranger.Payload, "key" | "name"> {
}
export declare const useRename: Flux.UseUpdate<RenameParams, RenameParams, z.ZodNever>;
export interface KeyParams {
    key: ranger.Key;
}
export declare const useName: Flux.UseSelect<RetrieveQuery, string>;
//# sourceMappingURL=queries.d.ts.map