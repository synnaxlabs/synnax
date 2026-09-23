import { type label } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/lyra/haul";
export declare const HAUL_TYPE = "label";
export type HaulItem = Haul.Item<typeof HAUL_TYPE, label.Key, undefined>;
export declare const createHaulItem: (key: label.Key) => HaulItem;
export declare const isHaulItem: (item: Haul.Item) => item is HaulItem;
export declare const filterHaulItems: (items: Haul.Item[]) => HaulItem[];
export declare const canDropHaulItem: Haul.CanDrop;
//# sourceMappingURL=types.d.ts.map