import { type ontology, panel, project } from "@synnaxlabs/client";
import { type optional, type record } from "@synnaxlabs/x";
import { type z } from "zod";
import { Flux } from "../flux";
export type RetrieveQuery = {
    key: panel.Key;
};
export declare const use: Flux.Use<RetrieveQuery, panel.Panel>, useEnsure: Flux.UseEnsure<RetrieveQuery>, useInvalidate: Flux.UseInvalidate<RetrieveQuery>, useResult: Flux.UseResult<RetrieveQuery, panel.Panel>, createSelector: Flux.CreateSelector<RetrieveQuery, panel.Panel>;
export type RetrieveKeysByProjectQuery = {
    project: project.Key;
};
export declare const useKeysByProject: Flux.Use<RetrieveKeysByProjectQuery, string[]>;
export interface KeyParams {
    key: panel.Key;
}
/**
 * Reports whether the subject may restructure the panel. Every mosaic gesture, from
 * closing a tab to resizing a split, reduces to one dispatch against the panel
 * document, so a single update grant answers all of them.
 */
export declare const useCanEdit: (arg?: optional.Optional<KeyParams, "key"> | undefined) => boolean;
export interface TabContentParams {
    key: panel.Key;
    tabKey: panel.TabKey;
}
export interface NodeParams extends KeyParams {
    nodeKey: number;
}
export declare const useNodeVariant: (arg: optional.Optional<NodeParams, "key">) => "leaf" | "split";
export declare const useLeafNode: (arg: optional.Optional<NodeParams, "key">) => Omit<panel.LeafNode, "tabs"> & {
    tabs: panel.TabKey[];
};
export declare const useSplitNode: (arg: optional.Optional<NodeParams, "key">) => panel.SplitNode;
export declare const useTabKeys: (arg?: optional.Optional<RetrieveQuery, "key"> | undefined) => string[];
export declare const useRoot: (arg?: optional.Optional<RetrieveQuery, "key"> | undefined) => {
    variant: "leaf";
    tabs: ({
        key: string;
        variant: "resource";
        resource: {
            type: "arc" | "builtin" | "channel" | "device" | "framer" | "group" | "label" | "lineplot" | "log" | "node" | "panel" | "policy" | "project" | "rack" | "range" | "range-alias" | "role" | "schematic" | "schematic_symbol" | "status" | "table" | "task" | "user" | "view";
            key: string;
        } | {
            type: "arc" | "builtin" | "channel" | "device" | "framer" | "group" | "label" | "lineplot" | "log" | "node" | "panel" | "policy" | "project" | "rack" | "range" | "range-alias" | "role" | "schematic" | "schematic_symbol" | "status" | "table" | "task" | "user" | "view";
            key: string;
        };
    } | {
        type: string;
        args: Record<string | number, unknown>;
        key: string;
        variant: "view";
    })[];
} | {
    variant: "split";
    direction: "x" | "y";
    size: number;
    first: panel.Node;
    last: panel.Node;
};
export declare const useName: (arg?: optional.Optional<RetrieveQuery, "key"> | undefined) => string;
export declare const useTab: (arg?: optional.Optional<TabContentParams, "key" | "tabKey"> | undefined) => panel.Tab;
export declare const useTabLeaf: (arg?: optional.Optional<TabContentParams, "key" | "tabKey"> | undefined) => panel.LeafNode;
export declare const useTabVariant: (arg?: optional.Optional<TabContentParams, "key" | "tabKey"> | undefined) => "resource" | "view";
export declare const useTabType: (arg?: optional.Optional<TabContentParams, "key" | "tabKey"> | undefined) => string;
export declare const useTabResource: (arg?: optional.Optional<TabContentParams, "key" | "tabKey"> | undefined) => {
    type: "arc" | "builtin" | "channel" | "device" | "framer" | "group" | "label" | "lineplot" | "log" | "node" | "panel" | "policy" | "project" | "rack" | "range" | "range-alias" | "role" | "schematic" | "schematic_symbol" | "status" | "table" | "task" | "user" | "view";
    key: string;
} | {
    type: "arc" | "builtin" | "channel" | "device" | "framer" | "group" | "label" | "lineplot" | "log" | "node" | "panel" | "policy" | "project" | "rack" | "range" | "range-alias" | "role" | "schematic" | "schematic_symbol" | "status" | "table" | "task" | "user" | "view";
    key: string;
};
export declare const useTabArgs: (arg?: optional.Optional<TabContentParams, "key" | "tabKey"> | undefined) => record.Unknown;
export declare const createSelectTabArgs: <Z extends z.ZodType>(schema: Z) => (() => z.output<Z>);
export interface ListParams extends Pick<panel.RetrieveRequest, "offset" | "limit"> {
}
export declare const useList: Flux.UseList<ListParams, string, panel.Panel>;
export interface CreateParams extends panel.New {
}
export declare const useCreate: Flux.UseUpdate<CreateParams, panel.Panel, z.ZodNever>;
export interface RenameParams extends Pick<panel.Panel, "key" | "name"> {
}
export declare const useRename: Flux.UseUpdate<RenameParams, RenameParams, z.ZodNever>;
export type DeleteParams = panel.Key | panel.Key[];
export declare const useDelete: Flux.UseUpdate<DeleteParams, DeleteParams, z.ZodNever>;
export declare const useDispatch: () => Flux.UseDispatchReturn<string, panel.Action>, useSingleDispatchBase: (params: record.Keyed<string>) => (action: panel.Action[] | panel.Action) => void, useUndoBase: (params: {
    key: string;
}) => Flux.UseUndoReturn, useRedoBase: (params: record.Keyed<string>) => Flux.UseRedoReturn;
export declare const useSingleDispatch: (arg?: optional.Optional<record.Keyed<string>, "key"> | undefined) => (action: panel.Action[] | panel.Action) => void;
export declare const useUndo: (arg?: optional.Optional<{
    key: string;
}, "key"> | undefined) => Flux.UseUndoReturn;
export declare const useRedo: (arg?: optional.Optional<record.Keyed<string>, "key"> | undefined) => Flux.UseRedoReturn;
export declare const useCloseResourceTabs: () => ((ids: ontology.ID | ontology.ID[]) => void);
export interface MoveTabToPanelParams extends Pick<panel.InsertTabsPayload, "targetLeaf" | "index" | "location"> {
    /** Panel currently holding the tab. */
    source: panel.Key;
    /** Panel the tab moves into. Must differ from source. */
    destination: panel.Key;
    tab: panel.Tab;
}
/**
 * useMoveTabToPanel moves a tab between two panels. The panels are separate documents,
 * so the move is two dispatches: the insert lands first and the source only gives the
 * tab up once it has, leaving the tab where it was if the destination rejects it.
 * @returns a callback resolving with the tab's key in the destination, or undefined
 * when the tab is not there.
 */
export declare const useMoveTabToPanel: () => ((params: MoveTabToPanelParams) => Promise<panel.TabKey | undefined>);
export declare const useSetCurrentTabResource: () => ((resource: ontology.ID) => void);
export declare const useSetCurrentTabView: () => ((view: panel.View) => void);
//# sourceMappingURL=queries.d.ts.map