import { Haul } from "@synnaxlabs/lyra/haul";
import { type Node } from "./node";
import { type AddNodeProps } from "./queries";
export declare const HAUL_TYPE = "schematic-element";
export interface HaulItemData<V extends Node.Variant = Node.Variant> extends AddNodeProps<V> {
}
export type HaulItem = Haul.Item<typeof HAUL_TYPE, string, HaulItemData>;
export declare const createHaulItem: <V extends Node.Variant>(data: HaulItemData<V>) => HaulItem;
export declare const isHaulItem: (item: Haul.Item) => item is HaulItem;
export declare const filterHaulItems: (items: Haul.Item[]) => HaulItem[];
export declare const canDropHaulItem: Haul.CanDrop;
//# sourceMappingURL=haul.d.ts.map