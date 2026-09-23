import { view } from "@synnaxlabs/client";
import { type z } from "zod";
import { Flux } from "../flux";
export type ListQuery = view.RetrieveMultipleParams;
export declare const useList: Flux.UseList<view.RetrieveMultipleParams, string, view.View>;
export declare const useCreate: Flux.UseUpdate<view.New, view.New, z.ZodNever>;
export type DeleteParams = view.Key | view.Key[];
export declare const useDelete: Flux.UseUpdate<DeleteParams, DeleteParams, z.ZodNever>;
export declare const formSchema: z.ZodObject<{
    key: z.ZodOptional<z.ZodDefault<z.ZodUUID>>;
    name: z.ZodString;
    type: z.ZodString;
    query: z.ZodDefault<z.ZodRecord<z.ZodUnion<readonly [z.ZodString, z.ZodNumber]>, z.ZodUnknown>>;
}, z.core.$strip>;
export type FormQuery = view.RetrieveSingleParams;
export declare const useForm: Flux.UseForm<view.RetrieveSingleParams, z.ZodObject<{
    key: z.ZodOptional<z.ZodDefault<z.ZodUUID>>;
    name: z.ZodString;
    type: z.ZodString;
    query: z.ZodDefault<z.ZodRecord<z.ZodUnion<readonly [z.ZodString, z.ZodNumber]>, z.ZodUnknown>>;
}, z.core.$strip>>;
export interface RenameParams extends Pick<view.View, "key" | "name"> {
}
export declare const useRename: Flux.UseUpdate<RenameParams, RenameParams, z.ZodNever>;
export declare const useSetSynchronizer: (onSet: (view: view.View) => void) => void;
export declare const useDeleteSynchronizer: (onDelete: (key: view.Key) => void) => void;
//# sourceMappingURL=queries.d.ts.map