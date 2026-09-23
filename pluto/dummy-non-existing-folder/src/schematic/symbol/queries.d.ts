import { type group, ontology, schematic } from "@synnaxlabs/client";
import type z from "zod";
import { Flux } from "../../flux";
export type RetrieveQuery = {
    key: string;
};
export declare const use: Flux.Use<RetrieveQuery, schematic.symbol.Symbol>, useResult: Flux.UseResult<RetrieveQuery, schematic.symbol.Symbol>;
export interface Resolved {
    symbol?: schematic.symbol.Symbol;
    missing: boolean;
}
/**
 * Resolves a symbol specification by key, kept live across edits and deletes.
 * Loading and missing are distinct: `symbol` is unset for both, `missing` is
 * true only once the reference is known to be dangling.
 */
export declare const useResolved: (key: string | null) => Resolved;
export type ListQuery = {
    keys?: string[];
    parent?: ontology.ID;
    searchTerm?: string;
    offset?: number;
    limit?: number;
};
export declare const useList: Flux.UseList<ListQuery, string, schematic.symbol.Symbol>;
export type FormQuery = {
    key: string;
};
export declare const formSchema: z.ZodObject<{
    key: z.ZodOptional<z.ZodDefault<z.ZodUUID>>;
    name: z.ZodString;
    data: z.ZodObject<{
        svg: z.ZodString;
        states: z.ZodDefault<z.ZodArray<z.ZodObject<{
            key: z.ZodString;
            name: z.ZodString;
            regions: z.ZodDefault<z.ZodArray<z.ZodObject<{
                key: z.ZodString;
                name: z.ZodString;
                selectors: z.ZodDefault<z.ZodArray<z.ZodString>>;
                strokeColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
                fillColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            }, z.core.$strip>>>;
        }, z.core.$strip>>>;
        variant: z.ZodString;
        handles: z.ZodDefault<z.ZodArray<z.ZodObject<{
            key: z.ZodString;
            position: z.ZodObject<{
                x: z.ZodNumber;
                y: z.ZodNumber;
            }, z.core.$strip>;
            orientation: z.ZodEnum<{
                bottom: "bottom";
                left: "left";
                right: "right";
                top: "top";
            }>;
        }, z.core.$strip>>>;
        scale: z.ZodDefault<z.ZodNumber>;
        strokeScaled: z.ZodDefault<z.ZodBoolean>;
        previewViewport: z.ZodOptional<z.ZodObject<{
            zoom: z.ZodDefault<z.ZodNumber>;
            position: z.ZodObject<{
                x: z.ZodNumber;
                y: z.ZodNumber;
            }, z.core.$strip>;
        }, z.core.$strip>>;
    }, z.core.$strip>;
    parent: z.ZodUnion<[z.ZodObject<{
        type: z.ZodEnum<{
            arc: "arc";
            builtin: "builtin";
            channel: "channel";
            device: "device";
            framer: "framer";
            group: "group";
            label: "label";
            lineplot: "lineplot";
            log: "log";
            node: "node";
            panel: "panel";
            policy: "policy";
            project: "project";
            rack: "rack";
            range: "range";
            "range-alias": "range-alias";
            role: "role";
            schematic: "schematic";
            schematic_symbol: "schematic_symbol";
            status: "status";
            table: "table";
            task: "task";
            user: "user";
            view: "view";
        }>;
        key: z.ZodString;
    }, z.core.$strip>, z.ZodPipe<z.ZodString, z.ZodTransform<{
        type: "arc" | "builtin" | "channel" | "device" | "framer" | "group" | "label" | "lineplot" | "log" | "node" | "panel" | "policy" | "project" | "rack" | "range" | "range-alias" | "role" | "schematic" | "schematic_symbol" | "status" | "table" | "task" | "user" | "view";
        key: string;
    }, string>>]>;
}, z.core.$strip>;
export declare const useForm: Flux.UseForm<FormQuery, z.ZodObject<{
    key: z.ZodOptional<z.ZodDefault<z.ZodUUID>>;
    name: z.ZodString;
    data: z.ZodObject<{
        svg: z.ZodString;
        states: z.ZodDefault<z.ZodArray<z.ZodObject<{
            key: z.ZodString;
            name: z.ZodString;
            regions: z.ZodDefault<z.ZodArray<z.ZodObject<{
                key: z.ZodString;
                name: z.ZodString;
                selectors: z.ZodDefault<z.ZodArray<z.ZodString>>;
                strokeColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
                fillColor: z.ZodOptional<z.ZodPipe<z.ZodUnion<readonly [z.ZodString, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt], null>, z.ZodTuple<[z.ZodInt, z.ZodInt, z.ZodInt, z.ZodNumber], null>, z.ZodTuple<[z.ZodNumber, z.ZodNumber, z.ZodNumber, z.ZodNumber], null>, z.ZodObject<{
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
            }, z.core.$strip>>>;
        }, z.core.$strip>>>;
        variant: z.ZodString;
        handles: z.ZodDefault<z.ZodArray<z.ZodObject<{
            key: z.ZodString;
            position: z.ZodObject<{
                x: z.ZodNumber;
                y: z.ZodNumber;
            }, z.core.$strip>;
            orientation: z.ZodEnum<{
                bottom: "bottom";
                left: "left";
                right: "right";
                top: "top";
            }>;
        }, z.core.$strip>>>;
        scale: z.ZodDefault<z.ZodNumber>;
        strokeScaled: z.ZodDefault<z.ZodBoolean>;
        previewViewport: z.ZodOptional<z.ZodObject<{
            zoom: z.ZodDefault<z.ZodNumber>;
            position: z.ZodObject<{
                x: z.ZodNumber;
                y: z.ZodNumber;
            }, z.core.$strip>;
        }, z.core.$strip>>;
    }, z.core.$strip>;
    parent: z.ZodUnion<[z.ZodObject<{
        type: z.ZodEnum<{
            arc: "arc";
            builtin: "builtin";
            channel: "channel";
            device: "device";
            framer: "framer";
            group: "group";
            label: "label";
            lineplot: "lineplot";
            log: "log";
            node: "node";
            panel: "panel";
            policy: "policy";
            project: "project";
            rack: "rack";
            range: "range";
            "range-alias": "range-alias";
            role: "role";
            schematic: "schematic";
            schematic_symbol: "schematic_symbol";
            status: "status";
            table: "table";
            task: "task";
            user: "user";
            view: "view";
        }>;
        key: z.ZodString;
    }, z.core.$strip>, z.ZodPipe<z.ZodString, z.ZodTransform<{
        type: "arc" | "builtin" | "channel" | "device" | "framer" | "group" | "label" | "lineplot" | "log" | "node" | "panel" | "policy" | "project" | "rack" | "range" | "range-alias" | "role" | "schematic" | "schematic_symbol" | "status" | "table" | "task" | "user" | "view";
        key: string;
    }, string>>]>;
}, z.core.$strip>>;
export interface RenameParams extends Pick<schematic.symbol.Symbol, "key" | "name"> {
}
export declare const useRename: Flux.UseUpdate<RenameParams, RenameParams, z.ZodNever>;
export type DeleteParams = schematic.symbol.Key | schematic.symbol.Key[];
export declare const useDelete: Flux.UseUpdate<DeleteParams, DeleteParams, z.ZodNever>;
export type DeleteGroupParams = group.Key;
export declare const useDeleteGroup: Flux.UseUpdate<string, string, z.ZodNever>;
export declare const useGroup: Flux.Use<{}, group.Group>, useResultGroup: Flux.UseResult<{}, group.Group>;
//# sourceMappingURL=queries.d.ts.map