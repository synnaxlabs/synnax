import { Haul } from "@synnaxlabs/lyra/haul";
export declare const HAUL_TYPE = "arc_element";
export type HaulItem = Haul.Item<typeof HAUL_TYPE, string, undefined>;
export declare const createHaulItem: (key: string) => HaulItem;
export declare const filterHaulItems: (items: Haul.Item[]) => HaulItem[];
export declare const canDropHaulItem: Haul.CanDrop;
//# sourceMappingURL=haul.d.ts.map