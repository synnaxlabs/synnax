import { type channel, type log, type project } from "@synnaxlabs/client";
import { Flux } from "../flux";
export type UseDeleteParams = log.Key | log.Key[];
export type RetrieveQuery = log.RetrieveSingleParams;
export declare const use: Flux.Use<{
    key: string;
}, log.Log>, useEnsure: Flux.UseEnsure<{
    key: string;
}>, useTombstone: Flux.UseTombstone<{
    key: string;
}>, createSelector: Flux.CreateSelector<{
    key: string;
}, log.Log>;
export interface KeyParams {
    key: log.Key;
}
export declare const useName: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => string;
export declare const useChannels: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => {
    channel: number;
    color: [number, number, number, number];
    notation: "engineering" | "scientific" | "standard";
    precision: number;
    alias: string;
    timestamp: {
        format: "ISO" | "ISODate" | "date" | "dateTime" | "preciseDate" | "preciseTime" | "time";
        tz: "UTC" | "local";
    };
}[];
export declare const useChannelKeys: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => number[];
export interface ChannelEntryParams extends KeyParams {
    channel: channel.Key;
}
export declare const useChannelEntry: (arg: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<ChannelEntryParams, "key">) => log.ChannelEntry | null;
export declare const useTimestampPrecision: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => number;
export declare const useIsHidingChannelNames: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => boolean;
export declare const useIsHidingReceiptTimestamp: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => boolean;
export declare const useDispatch: () => Flux.UseDispatchReturn<string, log.Action>, useUndoBase: (params: {
    key: string;
}) => Flux.UseUndoReturn, useRedoBase: (params: import("@synnaxlabs/x/dist/src/record/record.js").Keyed<string>) => Flux.UseRedoReturn, useSingleDispatchBase: (params: import("@synnaxlabs/x/dist/src/record/record.js").Keyed<string>) => (action: log.Action[] | log.Action) => void;
export declare const useSingleDispatch: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<import("@synnaxlabs/x/dist/src/record/record.js").Keyed<string>, "key"> | undefined) => (action: log.Action[] | log.Action) => void;
export declare const useUndo: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<{
    key: string;
}, "key"> | undefined) => Flux.UseUndoReturn;
export declare const useRedo: (arg?: import("@synnaxlabs/x/dist/src/optional/optional.js").Optional<import("@synnaxlabs/x/dist/src/record/record.js").Keyed<string>, "key"> | undefined) => Flux.UseRedoReturn;
export declare const useDelete: Flux.UseUpdate<UseDeleteParams, UseDeleteParams, import("zod").ZodNever>;
export interface CreateParams extends log.New {
    project?: project.Key;
}
export declare const useCreate: Flux.UseUpdate<CreateParams, log.Log, import("zod").ZodNever>;
export interface RenameParams extends Pick<log.Log, "key" | "name"> {
}
export declare const useRename: Flux.UseUpdate<RenameParams, RenameParams, import("zod").ZodNever>;
//# sourceMappingURL=queries.d.ts.map