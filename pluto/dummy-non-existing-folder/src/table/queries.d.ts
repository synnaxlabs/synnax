import { type project, table } from "@synnaxlabs/client";
import { type xy } from "@synnaxlabs/x";
import { Flux } from "../flux";
export type RetrieveQuery = table.RetrieveSingleParams;
export declare const use: Flux.Use<{
    key: string;
}, table.Table>, useEnsure: Flux.UseEnsure<{
    key: string;
}>, useTombstone: Flux.UseTombstone<{
    key: string;
}>, createSelector: Flux.CreateSelector<{
    key: string;
}, table.Table>;
export interface KeyParams {
    key: table.Key;
}
export declare const useName: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => string;
export declare const useRows: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => {
    size: number;
    cells: string[];
}[];
export declare const useColumns: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => {
    size: number;
}[];
export interface CellParams extends KeyParams {
    cellKey: string;
}
export declare const useCell: (arg: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<CellParams, "key">) => table.CellConfig | undefined;
export interface CellsParams extends KeyParams {
    cellKeys: string[];
}
export declare const useCells: (arg: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<CellsParams, "key">) => Map<string, table.CellConfig>;
export type DeleteParams = table.Key | table.Key[];
export declare const useDelete: Flux.UseUpdate<DeleteParams, DeleteParams, import("zod").ZodNever>;
export interface CreateParams extends table.New {
    project?: project.Key;
}
declare const useCreateBase: Flux.UseUpdate<CreateParams, table.Table, import("zod").ZodNever>;
export declare const useCreate: typeof useCreateBase;
export interface UseRenameParams {
    key: table.Key;
    name: string;
}
export declare const useRename: Flux.UseUpdate<UseRenameParams, UseRenameParams, import("zod").ZodNever>;
export declare const useDispatch: () => Flux.UseDispatchReturn<string, table.Action>, useUndoBase: (params: {
    key: string;
}) => Flux.UseUndoReturn, useRedoBase: (params: import("@synnaxlabs/x/dist/src/record/record.js").Keyed<string>) => Flux.UseRedoReturn, useSingleDispatchBase: (params: import("@synnaxlabs/x/dist/src/record/record.js").Keyed<string>) => (action: table.Action[] | table.Action) => void;
export declare const useSingleDispatch: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<import("@synnaxlabs/x/dist/src/record/record.js").Keyed<string>, "key"> | undefined) => (action: table.Action[] | table.Action) => void;
export declare const useUndo: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => Flux.UseUndoReturn;
export declare const useRedo: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<import("@synnaxlabs/x/dist/src/record/record.js").Keyed<string>, "key"> | undefined) => Flux.UseRedoReturn;
export declare const findCellPosition: (rows: table.Row[], cellKey: string) => xy.XY | null;
export declare const nextCellPosition: (rows: table.Row[], pos: xy.XY, dir: 1 | -1) => xy.XY | null;
export declare const cellsInRegion: (rows: table.Row[], start: xy.XY, end: xy.XY) => string[];
export declare const useCellPosition: (arg: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<CellParams, "key">) => xy.XY | null;
export {};
//# sourceMappingURL=queries.d.ts.map