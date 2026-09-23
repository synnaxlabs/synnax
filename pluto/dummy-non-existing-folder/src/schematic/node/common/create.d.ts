import { type schematic } from "@synnaxlabs/client";
import { type FC } from "react";
import { type Primitive as BasePrimitive } from "./primitive";
import { type Spec } from "../spec";
export interface SymbolParams<V extends schematic.NodeConfigType> {
    variant: V;
    name: string;
    label?: string;
    Primitive: FC<BasePrimitive.SVGBasedProps>;
    zIndex?: number;
}
export interface StaticConfig<V extends schematic.NodeConfigType> extends schematic.StaticSymbolConfig {
    variant: V;
}
export interface ToggleSymbolConfig<V extends schematic.NodeConfigType> extends schematic.ToggleSymbolConfig {
    variant: V;
}
export interface DummyToggleConfig<V extends schematic.NodeConfigType> extends schematic.DummyToggleSymbolConfig {
    variant: V;
}
export declare const createStatic: <V extends schematic.NodeConfigType>({ variant, name, label, Primitive, zIndex, }: SymbolParams<V>) => {
    spec: Spec<V, StaticConfig<V>>;
};
interface ToggleParams<V extends schematic.NodeConfigType> extends SymbolParams<V> {
    node?: "toggle" | "labeled";
}
export declare const createToggle: <V extends schematic.NodeConfigType>({ variant, name, label, Primitive, zIndex, node, }: ToggleParams<V>) => {
    spec: Spec<V, ToggleSymbolConfig<V>>;
};
export declare const createDummyToggle: <V extends schematic.NodeConfigType>({ variant, name, label, Primitive, zIndex, }: SymbolParams<V>) => {
    spec: Spec<V, DummyToggleConfig<V>>;
};
export {};
//# sourceMappingURL=create.d.ts.map