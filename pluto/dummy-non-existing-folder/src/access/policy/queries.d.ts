import { access } from "@synnaxlabs/client";
import { type List } from "@synnaxlabs/lyra/list";
import { z } from "zod";
import { Flux } from "../../flux";
export declare const formSchema: z.ZodObject<{
    key: z.ZodOptional<z.ZodUUID>;
    name: z.ZodString;
    objects: z.ZodArray<z.ZodUnion<[z.ZodObject<{
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
    }, string>>]>>;
    actions: z.ZodArray<z.ZodEnum<{
        create: "create";
        delete: "delete";
        retrieve: "retrieve";
        update: "update";
    }>>;
}, z.core.$strip>;
export type RetrieveQuery = {
    key: access.policy.Key;
};
export declare const use: Flux.Use<RetrieveQuery, access.policy.Policy>;
export type ListParams = List.PagerParams;
export declare const useList: Flux.UseList<List.PagerParams, string, access.policy.Policy>;
export type DeleteParams = access.policy.Key | access.policy.Key[];
export declare const useDelete: Flux.UseUpdate<DeleteParams, DeleteParams, z.ZodNever>;
export interface RenameParams {
    key: access.policy.Key;
    name: string;
}
export declare const useRename: Flux.UseUpdate<RenameParams, RenameParams, z.ZodNever>;
export declare const useForm: Flux.UseForm<RetrieveQuery, z.ZodObject<{
    key: z.ZodOptional<z.ZodUUID>;
    name: z.ZodString;
    objects: z.ZodArray<z.ZodUnion<[z.ZodObject<{
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
    }, string>>]>>;
    actions: z.ZodArray<z.ZodEnum<{
        create: "create";
        delete: "delete";
        retrieve: "retrieve";
        update: "update";
    }>>;
}, z.core.$strip>>;
//# sourceMappingURL=queries.d.ts.map