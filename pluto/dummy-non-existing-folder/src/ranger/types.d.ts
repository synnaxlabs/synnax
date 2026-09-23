import { type ranger } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/lyra/haul";
import { type NumericTimeRange } from "@synnaxlabs/x";
export declare const HAUL_TYPE = "range";
export interface HaulData {
    key: ranger.Key;
    name: string;
    timeRange: NumericTimeRange;
}
export type HaulItem = Haul.Item<typeof HAUL_TYPE, ranger.Key, HaulData>;
export declare const createHaulItem: ({ key, name, timeRange }: ranger.Payload) => HaulItem;
export declare const isHaulItem: (item: Haul.Item) => item is HaulItem;
export declare const filterHaulItems: (items: Haul.Item[]) => HaulItem[];
export declare const canDropHaulItem: Haul.CanDrop;
//# sourceMappingURL=types.d.ts.map