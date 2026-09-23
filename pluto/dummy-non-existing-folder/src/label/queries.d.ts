import { label, type ontology, type Synnax as Client } from "@synnaxlabs/client";
import type z from "zod";
import { Flux } from "../flux";
export type RetrieveQuery = label.RetrieveSingleParams;
export type LabelsOfQuery = {
    id: ontology.ID;
};
interface SetLabelsForParams {
    client: Client;
    data: {
        id: ontology.ID;
        labels: label.Key[];
    };
}
export declare const setLabelsFor: ({ client, data: { id, labels }, }: SetLabelsForParams) => Promise<label.Label[]>;
export declare const useResultLabelsOf: Flux.UseResult<LabelsOfQuery, label.Label[]>;
export type ListQuery = label.RetrieveMultipleParams;
export declare const useList: Flux.UseList<{
    keys?: string[] | undefined;
    names?: string[] | undefined;
    for?: string | {
        type: "arc" | "builtin" | "channel" | "device" | "framer" | "group" | "label" | "lineplot" | "log" | "node" | "panel" | "policy" | "project" | "rack" | "range" | "range-alias" | "role" | "schematic" | "schematic_symbol" | "status" | "table" | "task" | "user" | "view";
        key: string;
    } | undefined;
    searchTerm?: string | undefined;
    offset?: number | undefined;
    limit?: number | undefined;
    ignoreNotFoundError?: boolean | undefined;
}, string, label.Label>;
type FormQuery = {
    key: label.Key;
};
export declare const formSchema: z.ZodObject<{
    key: z.ZodOptional<z.ZodDefault<z.ZodUUID>>;
    name: z.ZodString;
    color: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
    } | [number, number, number] | [number, number, number, number]>>;
}, z.core.$strip>;
export declare const useForm: Flux.UseForm<FormQuery, z.ZodObject<{
    key: z.ZodOptional<z.ZodDefault<z.ZodUUID>>;
    name: z.ZodString;
    color: z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
    } | [number, number, number] | [number, number, number, number]>>;
}, z.core.$strip>>;
export type DeleteParams = label.Key | label.Key[];
export declare const useDelete: Flux.UseUpdate<DeleteParams, DeleteParams, z.ZodNever>;
export type RetrieveMultipleParams = {
    keys: label.Key[];
};
export declare const useMultiple: Flux.Use<RetrieveMultipleParams, label.Label[]>, useResultMultiple: Flux.UseResult<RetrieveMultipleParams, label.Label[]>;
export {};
//# sourceMappingURL=queries.d.ts.map