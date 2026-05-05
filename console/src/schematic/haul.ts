import { type Haul, type Schematic } from "@synnaxlabs/pluto";

import { type AddNodeProps } from "@/schematic/symbols/useAddNode";

export const SYMBOL_HAUL_TYPE = "schematic_symbol";

export interface SymbolHaulData extends AddNodeProps {}

export type SymbolHaulItem = Haul.Item<typeof SYMBOL_HAUL_TYPE, string, SymbolHaulData>;

export const createSymbolHaulItem = (data: SymbolHaulData): SymbolHaulItem => ({
  type: SYMBOL_HAUL_TYPE,
  key: data.key,
  data,
});

export const isSymbolHaulItem = (item: Haul.Item): item is SymbolHaulItem =>
  item.type === SYMBOL_HAUL_TYPE;

export const filterSymbolHaulItems = (items: Haul.Item[]): SymbolHaulItem[] =>
  items.filter(isSymbolHaulItem);

export const VALUE_HAUL_TYPE = "schematic_value";

export type ValueHaulData = Schematic.Node.ConfigOf<"value">;

export type ValueHaulItem = Haul.Item<typeof VALUE_HAUL_TYPE, string, ValueHaulData>;

export const createValueHaulItem = (props: ValueHaulData): ValueHaulItem => ({
  type: VALUE_HAUL_TYPE,
  key: "value",
  data: props,
});

export const isValueHaulItem = (item: Haul.Item): item is ValueHaulItem =>
  item.type === VALUE_HAUL_TYPE;

export const filterValueHaulItems = (items: Haul.Item[]): ValueHaulItem[] =>
  items.filter(isValueHaulItem);

export const isHaulItem = (item: Haul.Item): item is SymbolHaulItem | ValueHaulItem =>
  isSymbolHaulItem(item) || isValueHaulItem(item);

export const canDropHaulItem: Haul.CanDrop = ({ items }) => items.some(isHaulItem);
