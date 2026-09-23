import { type group } from "@synnaxlabs/client";
import { type Icon } from "@synnaxlabs/lyra/icon";
import { type Variant } from "./registry";
export interface Group extends group.Group {
    Icon: Icon.FC;
    symbols: Variant[];
}
export declare const GROUPS: Group[];
//# sourceMappingURL=group.d.ts.map