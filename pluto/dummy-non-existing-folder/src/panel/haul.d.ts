import { panel } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";
import { z } from "zod";
declare const tabDragPayloadZ: z.ZodObject<{
    panel: z.ZodUUID;
    tab: z.ZodDiscriminatedUnion<[z.ZodObject<{
        key: z.ZodDefault<z.ZodUUID>;
        variant: z.ZodLiteral<"resource">;
        resource: z.ZodUnion<[z.ZodObject<{
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
    }, z.core.$strip>, z.ZodObject<{
        type: z.ZodString;
        args: z.ZodDefault<z.ZodRecord<z.ZodUnion<readonly [z.ZodString, z.ZodNumber]>, z.ZodUnknown>>;
        key: z.ZodDefault<z.ZodUUID>;
        variant: z.ZodLiteral<"view">;
    }, z.core.$strip>], "variant">;
}, z.core.$strip>;
/** The origin of a dragged tab: the panel holding it and the tab itself. */
export interface TabDragPayload extends z.infer<typeof tabDragPayloadZ> {
}
/**
 * Builds the payload a dragged tab carries. The tab travels whole because the drop may
 * land in a window that has never loaded the source panel.
 */
export declare const createTabDragPayload: (key: panel.Key, tab: panel.Tab) => record.Unknown;
/** @returns the origin of a dragged tab, or undefined if data is not a tab payload. */
export declare const parseTabDragPayload: (data: unknown) => TabDragPayload | undefined;
export {};
//# sourceMappingURL=haul.d.ts.map