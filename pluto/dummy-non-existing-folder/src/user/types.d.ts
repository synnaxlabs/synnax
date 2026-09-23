import { type user } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/lyra/haul";
export declare const HAUL_TYPE = "user";
export type HaulItem = Haul.Item<typeof HAUL_TYPE, user.Key, user.User>;
export declare const createHaulItem: (payload: user.User) => HaulItem;
export declare const isHaulItem: (item: Haul.Item) => item is HaulItem;
export declare const filterHaulItems: (items: Haul.Item[]) => HaulItem[];
export declare const canDropHaulItem: Haul.CanDrop;
//# sourceMappingURL=types.d.ts.map