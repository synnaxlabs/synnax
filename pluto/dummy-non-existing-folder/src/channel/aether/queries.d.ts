import { type channel, type ranger } from "@synnaxlabs/client";
import { type flux } from "../../flux/aether";
export declare const RESOURCE_NAME = "channel";
export declare const PLURAL_RESOURCE_NAME = "channels";
export type RetrieveQuery = {
    key: channel.Key;
    rangeKey?: ranger.Key;
};
export declare const retrieveDefinition: flux.Definition<RetrieveQuery, channel.Channel>;
export type RetrieveMultipleQuery = channel.RetrieveOptions & {
    keys: channel.Key[];
};
export declare const retrieveMultipleDefinition: flux.Definition<RetrieveMultipleQuery, channel.Channel[]>;
//# sourceMappingURL=queries.d.ts.map