import { type channel, type control } from "@synnaxlabs/client";
import { type flux } from "../../../flux/aether";
export declare const RESOURCE_NAME = "control state";
export declare const PLURAL_RESOURCE_NAME = "control states";
export type RetrieveQuery = {
    key: channel.Key;
};
export declare const retrieveDefinition: flux.Definition<RetrieveQuery, control.KeyedState>;
export type ListQuery = {
    keys: channel.Key[];
};
export declare const listDefinition: flux.Definition<ListQuery, control.KeyedState[]>;
//# sourceMappingURL=queries.d.ts.map