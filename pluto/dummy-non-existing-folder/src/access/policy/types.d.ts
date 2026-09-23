import { type access } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/lyra/haul";
export declare const HAUL_TYPE = "policy";
export type HaulItem = Haul.Item<typeof HAUL_TYPE, access.policy.Key, undefined>;
export declare const createHaulItem: (key: access.policy.Key) => HaulItem;
export declare const isHaulItem: (item: Haul.Item) => item is HaulItem;
export declare const filterHaulItems: (items: Haul.Item[]) => HaulItem[];
export declare const canDropHaulItem: Haul.CanDrop;
//# sourceMappingURL=types.d.ts.map