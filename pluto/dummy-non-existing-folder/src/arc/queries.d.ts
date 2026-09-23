import { arc, type rack, status, task } from "@synnaxlabs/client";
import { type List } from "@synnaxlabs/lyra/list";
import { type record, xy } from "@synnaxlabs/x";
import z from "zod";
import { Flux } from "../flux";
declare const useDispatch: () => Flux.UseDispatchReturn<string, arc.Action>;
export { useDispatch };
export declare const useUndo: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => Flux.UseUndoReturn;
export declare const useRedo: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<record.Keyed<string>, "key"> | undefined) => Flux.UseRedoReturn;
export declare const useSingleDispatch: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<record.Keyed<string>, "key"> | undefined) => (action: arc.Action[] | arc.Action) => void;
export interface KeyParams {
    key: arc.Key;
}
export type RetrieveQuery = {
    key: arc.Key;
    includeStatus?: boolean;
};
export declare const use: Flux.Use<RetrieveQuery, arc.Arc>, useEnsure: Flux.UseEnsure<RetrieveQuery>, useTombstone: Flux.UseTombstone<RetrieveQuery>, useResult: Flux.UseResult<RetrieveQuery, arc.Arc>, createSelector: Flux.CreateSelector<RetrieveQuery, arc.Arc>;
export declare const useAllNodes: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<RetrieveQuery, "key"> | undefined) => {
    key: string;
    position: {
        x: number;
        y: number;
    };
    zIndex?: number | undefined;
    type?: string | undefined;
    draggable?: boolean | undefined;
}[];
export interface NodesParams extends KeyParams {
    keys: string[];
}
export declare const useNodes: (arg: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<NodesParams, "key">) => {
    key: string;
    position: {
        x: number;
        y: number;
    };
    zIndex?: number | undefined;
    type?: string | undefined;
    draggable?: boolean | undefined;
}[];
export declare const useAllEdges: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<RetrieveQuery, "key"> | undefined) => {
    key: string;
    source: {
        node: string;
        param: string;
    };
    target: {
        node: string;
        param: string;
    };
}[];
export interface NodePropsParams extends KeyParams {
    nodeKey: string;
}
export declare const useNodeConfig: (arg: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<NodePropsParams, "key">) => {
    type: "constant";
    value: number;
} | {
    type: "on";
    channel: number;
} | {
    type: "stable_for";
    duration: number;
} | {
    type: "add";
} | {
    type: "subtract";
} | {
    type: "multiply";
} | {
    type: "divide";
} | {
    type: "gt";
} | {
    type: "lt";
} | {
    type: "eq";
} | {
    type: "ne";
} | {
    type: "ge";
} | {
    type: "le";
} | {
    type: "and";
} | {
    type: "or";
} | {
    type: "not";
} | {
    type: "write";
    channel: number;
    value: number;
} | {
    type: "select";
} | {
    type: "status.set";
    key_or_name: string;
    variant: "disabled" | "error" | "info" | "loading" | "success" | "warning";
    message: string;
};
export declare const useMode: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<RetrieveQuery, "key"> | undefined) => "graph" | "text";
export declare const useHasText: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<RetrieveQuery, "key"> | undefined) => boolean;
export declare const useName: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<RetrieveQuery, "key"> | undefined) => string;
export interface AddNodeProps {
    key: string;
    type: string;
    position?: xy.Crude;
}
export declare const useAddNode: (keyOverride?: arc.Key) => ({ key: nodeKey, type, position }: AddNodeProps) => void;
export type ListQuery = List.PagerParams & {
    keys?: arc.Key[];
};
export declare const useList: Flux.UseList<ListQuery, string, arc.Arc>;
export declare const useDelete: Flux.UseUpdate<string | string[], string | string[], z.ZodNever>;
export declare const formSchema: z.ZodObject<{
    key: z.ZodOptional<z.ZodUUID>;
    name: z.ZodString;
    mode: z.ZodEnum<{
        graph: "graph";
        text: "text";
    }>;
}, z.core.$strip>;
export declare const ZERO_FORM_VALUES: z.infer<typeof formSchema>;
export type FormQuery = Record<string, never>;
export declare const useForm: Flux.UseForm<FormQuery, z.ZodObject<{
    key: z.ZodOptional<z.ZodUUID>;
    name: z.ZodString;
    mode: z.ZodEnum<{
        graph: "graph";
        text: "text";
    }>;
}, z.core.$strip>>;
export interface CreateParams extends arc.New {
}
export declare const useCreate: Flux.UseUpdate<CreateParams, arc.Arc, z.ZodNever>;
export interface RenameParams extends Pick<arc.Arc, "key" | "name"> {
}
export declare const useRename: Flux.UseUpdate<RenameParams, RenameParams, z.ZodNever>;
export interface SetRackParams {
    key: arc.Key;
    /** Target rack. Zero clears the binding, deleting the arc's task. */
    rack: rack.Key;
}
export declare const useSetRack: Flux.UseUpdate<SetRackParams, task.Task<task.PayloadSchemas<z.ZodType<string, unknown, z.core.$ZodTypeInternals<string, unknown>>, z.ZodType<record.Unknown, unknown, z.core.$ZodTypeInternals<record.Unknown, unknown>>, z.ZodType<unknown, unknown, z.core.$ZodTypeInternals<unknown, unknown>>>> | null, z.ZodNever>;
export type RetrieveTaskParams = {
    arcKey: arc.Key;
};
export declare const useTask: Flux.Use<RetrieveTaskParams, task.Task<task.PayloadSchemas<z.ZodType<string, unknown, z.core.$ZodTypeInternals<string, unknown>>, z.ZodType<record.Unknown, unknown, z.core.$ZodTypeInternals<record.Unknown, unknown>>, z.ZodType<unknown, unknown, z.core.$ZodTypeInternals<unknown, unknown>>>> | null>, useResultTask: Flux.UseResult<RetrieveTaskParams, task.Task<task.PayloadSchemas<z.ZodType<string, unknown, z.core.$ZodTypeInternals<string, unknown>>, z.ZodType<record.Unknown, unknown, z.core.$ZodTypeInternals<record.Unknown, unknown>>, z.ZodType<unknown, unknown, z.core.$ZodTypeInternals<unknown, unknown>>>> | null>, createTaskResultSelector: Flux.CreateResultSelector<RetrieveTaskParams, task.Task<task.PayloadSchemas<z.ZodType<string, unknown, z.core.$ZodTypeInternals<string, unknown>>, z.ZodType<record.Unknown, unknown, z.core.$ZodTypeInternals<record.Unknown, unknown>>, z.ZodType<unknown, unknown, z.core.$ZodTypeInternals<unknown, unknown>>>> | null>;
/**
 * Whether the arc's running instance was deployed from different content, config, or
 * rack than its task now holds. The Core rewrites the task config whenever the arc's
 * semantic content changes, so content drift surfaces as ordinary task config drift.
 * Arcs that are not running never drift. Re-renders only when the boolean changes.
 */
export declare const useDrifted: (query: RetrieveTaskParams) => boolean;
export interface UseTaskControlsReturn {
    running: boolean;
    taskKey: task.Key;
    taskRack: rack.Key;
    /** Starts the task. The driver rebuilds from the current config when it drifted. */
    onStart: () => void;
    /** Stops the running instance. */
    onStop: () => void;
    onStartStop: () => void;
    taskStatus: status.Status;
}
/** Running state and start/stop controls for the arc's task. */
export declare const useTaskControls: (key: arc.Key, name: string) => UseTaskControlsReturn;
//# sourceMappingURL=queries.d.ts.map