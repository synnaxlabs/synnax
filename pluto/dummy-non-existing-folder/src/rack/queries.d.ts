import { type rack } from "@synnaxlabs/client";
import { Flux } from "../flux";
export type RetrieveQuery = {
    key: rack.Key;
    includeStatus?: boolean;
};
export type ListQuery = rack.RetrieveMultipleParams;
export declare const useList: Flux.UseList<{
    keys?: number[] | undefined;
    names?: string[] | undefined;
    integration?: string | undefined;
    searchTerm?: string | undefined;
    embedded?: boolean | undefined;
    hostIsNode?: boolean | undefined;
    limit?: number | undefined;
    offset?: number | undefined;
    includeStatus?: boolean | undefined;
}, number, rack.Rack>;
export declare const use: Flux.Use<RetrieveQuery, rack.Rack>, useResult: Flux.UseResult<RetrieveQuery, rack.Rack>, useEnsure: Flux.UseEnsure<RetrieveQuery>, createSelector: Flux.CreateSelector<RetrieveQuery, rack.Rack>;
export declare const useName: Flux.UseSelect<RetrieveQuery, string>;
export declare const useIntegrations: Flux.UseSelect<RetrieveQuery, string[]>;
export type UseDeleteParams = rack.Key | rack.Key[];
export declare const useDelete: Flux.UseUpdate<UseDeleteParams, UseDeleteParams, import("zod").ZodNever>;
export interface RenameParams extends Pick<rack.Rack, "key" | "name"> {
}
export declare const useRename: Flux.UseUpdate<RenameParams, RenameParams, import("zod").ZodNever>;
//# sourceMappingURL=queries.d.ts.map