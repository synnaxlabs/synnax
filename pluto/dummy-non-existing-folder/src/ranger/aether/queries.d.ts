import { type ranger } from "@synnaxlabs/client";
import { type flux } from "../../flux/aether";
export type ListQuery = Omit<ranger.RetrieveRequest, "names">;
export declare const listDefinition: flux.Definition<ListQuery, ranger.Range[]>;
//# sourceMappingURL=queries.d.ts.map