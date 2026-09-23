import { device } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";
import { type z } from "zod";
import { Flux } from "../flux";
export declare const useSetSynchronizer: (onSet: (device: Omit<device.Device, "status">) => void) => void;
export type RetrieveQuery = device.RetrieveSingleParams;
export declare const createRetrieve: <Properties extends z.ZodType<record.Unknown> = z.ZodType<record.Unknown>, Make extends z.ZodType<string> = z.ZodString, Model extends z.ZodType<string> = z.ZodString>(schemas?: device.DeviceSchemas<Properties, Make, Model>) => Flux.CreateRetrieveReturn<{
    key: string;
    includeStatus?: boolean | undefined;
}, device.Device<Properties, Make, Model>>;
export declare const use: Flux.Use<{
    key: string;
    includeStatus?: boolean | undefined;
}, device.Device<z.ZodType<record.Unknown, unknown, z.core.$ZodTypeInternals<record.Unknown, unknown>>, z.ZodString, z.ZodString>>, useResult: Flux.UseResult<{
    key: string;
    includeStatus?: boolean | undefined;
}, device.Device<z.ZodType<record.Unknown, unknown, z.core.$ZodTypeInternals<record.Unknown, unknown>>, z.ZodString, z.ZodString>>, createResultSelector: Flux.CreateResultSelector<{
    key: string;
    includeStatus?: boolean | undefined;
}, device.Device<z.ZodType<record.Unknown, unknown, z.core.$ZodTypeInternals<record.Unknown, unknown>>, z.ZodString, z.ZodString>>;
/** Compared by variant and message: a heartbeat that changes neither is silenced. */
export declare const useResultStatus: Flux.UseResult<{
    key: string;
    includeStatus?: boolean | undefined;
}, {
    key: string;
    name: string;
    variant: "disabled" | "error" | "info" | "loading" | "success" | "warning";
    message: string;
    description: string;
    time: import("@synnaxlabs/x").TimeStamp;
    labels?: {
        key: string;
        name: string;
        color: [number, number, number, number];
    }[] | undefined;
    details: {
        rack: number;
        device: string;
    };
} | undefined>;
export declare const useResultRack: Flux.UseResult<{
    key: string;
    includeStatus?: boolean | undefined;
}, number>;
export type ListParams = device.RetrieveMultipleParams;
export declare const useList: Flux.UseList<{
    keys?: string[] | undefined;
    names?: string[] | undefined;
    makes?: string[] | undefined;
    models?: string[] | undefined;
    locations?: string[] | undefined;
    racks?: number[] | undefined;
    searchTerm?: string | undefined;
    limit?: number | undefined;
    offset?: number | undefined;
    includeStatus?: boolean | undefined;
    includeParent?: boolean | undefined;
}, string, device.Device<z.ZodType<record.Unknown, unknown, z.core.$ZodTypeInternals<record.Unknown, unknown>>, z.ZodString, z.ZodString>>;
export type UseDeleteParams = device.Key | device.Key[];
export declare const useDelete: Flux.UseUpdate<UseDeleteParams, UseDeleteParams, z.ZodNever>;
export declare const createCreate: <Properties extends z.ZodType<record.Unknown> = z.ZodType<record.Unknown>, Make extends z.ZodType<string> = z.ZodString, Model extends z.ZodType<string> = z.ZodString>(schemas?: device.DeviceSchemas<Properties, Make, Model>) => Flux.CreateUpdateReturn<device.New<Properties, Make, Model>, device.Device<Properties, Make, Model>, z.ZodNever>;
export declare const useCreate: Flux.UseUpdate<device.New<z.ZodType<record.Unknown, unknown, z.core.$ZodTypeInternals<record.Unknown, unknown>>, z.ZodString, z.ZodString>, device.Device<z.ZodType<record.Unknown, unknown, z.core.$ZodTypeInternals<record.Unknown, unknown>>, z.ZodString, z.ZodString>, z.ZodNever>;
export type UseRetrieveGroupParams = Record<string, never>;
export declare const useGroupID: Flux.Use<UseRetrieveGroupParams, {
    type: "arc" | "builtin" | "channel" | "device" | "framer" | "group" | "label" | "lineplot" | "log" | "node" | "panel" | "policy" | "project" | "rack" | "range" | "range-alias" | "role" | "schematic" | "schematic_symbol" | "status" | "table" | "task" | "user" | "view";
    key: string;
} | {
    type: "arc" | "builtin" | "channel" | "device" | "framer" | "group" | "label" | "lineplot" | "log" | "node" | "panel" | "policy" | "project" | "rack" | "range" | "range-alias" | "role" | "schematic" | "schematic_symbol" | "status" | "table" | "task" | "user" | "view";
    key: string;
} | undefined>;
export interface RenameParams extends Pick<device.Device, "key" | "name"> {
}
export declare const useRename: Flux.UseUpdate<RenameParams, RenameParams, z.ZodNever>;
export declare const formSchema: z.ZodObject<{
    key: z.ZodString;
    rack: z.ZodUInt32;
    location: z.ZodString;
    make: z.ZodString;
    model: z.ZodString;
    name: z.ZodString;
    configured: z.ZodDefault<z.ZodBoolean>;
    properties: z.ZodDefault<z.ZodRecord<z.ZodUnion<readonly [z.ZodString, z.ZodNumber]>, z.ZodUnknown>> | z.ZodType<record.Unknown, unknown, z.core.$ZodTypeInternals<record.Unknown, unknown>>;
    status: z.ZodOptional<import("@synnaxlabs/client").StatusZodObject<z.ZodObject<{
        rack: z.ZodUInt32;
        device: z.ZodString;
    }, z.core.$strip>, z.ZodEnum<{
        disabled: "disabled";
        error: "error";
        info: "info";
        loading: "loading";
        success: "success";
        warning: "warning";
    }>>>;
    parent: z.ZodOptional<z.ZodUnion<[z.ZodObject<{
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
}, z.core.$strip>;
export type FormQuery = RetrieveQuery;
export declare const createForm: <Properties extends z.ZodType<record.Unknown> = z.ZodType<record.Unknown>, Make extends z.ZodType<string> = z.ZodString, Model extends z.ZodType<string> = z.ZodString>(schemas?: device.DeviceSchemas<Properties, Make, Model>) => Flux.UseForm<{
    key: string;
    includeStatus?: boolean | undefined;
}, z.ZodObject<{
    key: z.ZodString;
    rack: z.ZodUInt32;
    location: z.ZodString;
    make: z.ZodString;
    model: z.ZodString;
    name: z.ZodString;
    configured: z.ZodDefault<z.ZodBoolean>;
    properties: z.ZodDefault<z.ZodRecord<z.ZodUnion<readonly [z.ZodString, z.ZodNumber]>, z.ZodUnknown>> | z.ZodType<record.Unknown, unknown, z.core.$ZodTypeInternals<record.Unknown, unknown>>;
    status: z.ZodOptional<import("@synnaxlabs/client").StatusZodObject<z.ZodObject<{
        rack: z.ZodUInt32;
        device: z.ZodString;
    }, z.core.$strip>, z.ZodEnum<{
        disabled: "disabled";
        error: "error";
        info: "info";
        loading: "loading";
        success: "success";
        warning: "warning";
    }>>>;
    parent: z.ZodOptional<z.ZodUnion<[z.ZodObject<{
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
}, z.core.$strip>>;
export declare const useForm: Flux.UseForm<{
    key: string;
    includeStatus?: boolean | undefined;
}, z.ZodObject<{
    key: z.ZodString;
    rack: z.ZodUInt32;
    location: z.ZodString;
    make: z.ZodString;
    model: z.ZodString;
    name: z.ZodString;
    configured: z.ZodDefault<z.ZodBoolean>;
    properties: z.ZodDefault<z.ZodRecord<z.ZodUnion<readonly [z.ZodString, z.ZodNumber]>, z.ZodUnknown>> | z.ZodType<record.Unknown, unknown, z.core.$ZodTypeInternals<record.Unknown, unknown>>;
    status: z.ZodOptional<import("@synnaxlabs/client").StatusZodObject<z.ZodObject<{
        rack: z.ZodUInt32;
        device: z.ZodString;
    }, z.core.$strip>, z.ZodEnum<{
        disabled: "disabled";
        error: "error";
        info: "info";
        loading: "loading";
        success: "success";
        warning: "warning";
    }>>>;
    parent: z.ZodOptional<z.ZodUnion<[z.ZodObject<{
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
}, z.core.$strip>>;
//# sourceMappingURL=queries.d.ts.map