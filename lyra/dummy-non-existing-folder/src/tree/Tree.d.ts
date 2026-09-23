import { type compare, type record, type state as xstate } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { type Component } from "../component";
import { Haul } from "../haul";
import { List } from "../list";
import { Select } from "../select";
import { type Node, type Shape } from "./base";
export declare const HAUL_TYPE = "tree_item";
export type HaulItem = Haul.Item<typeof HAUL_TYPE, string, undefined>;
export declare const createHaulItem: (key: string) => HaulItem;
export declare const isHaulItem: (item: Haul.Item) => item is HaulItem;
export declare const filterHaulItems: (items: Haul.Item[]) => HaulItem[];
export declare const canDropHaulItem: Haul.CanDrop;
export interface HandleExpandProps<K extends record.Key = string> {
    current: K[];
    action: "expand" | "contract";
    clicked: K;
}
export interface UseProps<K extends record.Key = string> {
    onExpand?: (props: HandleExpandProps<K>) => void;
    selected?: K[];
    sort?: compare.Comparator<Node<K>>;
    onSelectedChange?: xstate.Setter<K[]>;
    initialExpanded?: K[];
    nodes: Node<K>[];
}
export interface UseReturn<K extends record.Key = string> {
    selected: K[];
    onSelect: Select.UseMultipleProps<K>["onChange"];
    expanded: K[];
    expand: (key: K) => void;
    contract: (...keys: K[]) => void;
    clearExpanded: () => void;
    shape: Shape<K>;
}
export declare const use: <K extends record.Key = string>({ onExpand, nodes, initialExpanded, selected: propsSelected, onSelectedChange, sort, }: UseProps<K>) => UseReturn<K>;
export interface ItemRenderProps<K extends record.Key = string> extends List.ItemRenderProps<K> {
}
export interface TreeProps<K extends record.Key, E extends record.Keyed<K>> extends Omit<Select.FrameProps<K, E>, "children" | "ref" | "virtualizer" | "data" | "onChange">, Omit<List.ItemsProps<K>, "children" | "onSelect">, UseReturn<K> {
    children: Component.RenderProp<ItemRenderProps<K>>;
    showRules?: boolean;
    shape: Shape<K>;
}
export declare const Tree: <K extends record.Key, E extends record.Keyed<K>>({ shape, children, selected, onSelect, getItem, subscribe, className, contract: _, expand: __, expanded: ___, className: ____, clearExpanded: _____, showRules, virtual, itemHeight, overscan, onFetchMore, ...rest }: TreeProps<K, E>) => ReactElement;
//# sourceMappingURL=Tree.d.ts.map