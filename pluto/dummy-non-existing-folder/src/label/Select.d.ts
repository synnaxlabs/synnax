import { type label } from "@synnaxlabs/client";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";
import { type Flux } from "../flux";
import { type ListQuery } from "./queries";
export interface SelectMultipleProps extends Omit<Select.MultipleProps<label.Key, label.Label | undefined>, "data" | "resourceName" | "subscribe" | "children">, Flux.UseListParams<ListQuery, label.Key, label.Label> {
}
export declare const SelectMultiple: ({ filter, initialQuery, ...rest }: SelectMultipleProps) => ReactElement;
export interface SelectSingleProps extends Omit<Select.SingleProps<label.Key, label.Label | undefined>, "data" | "resourceName" | "subscribe" | "children">, Flux.UseListParams<ListQuery, label.Key, label.Label> {
}
export declare const SelectSingle: ({ filter, initialQuery, ...rest }: SelectSingleProps) => ReactElement;
//# sourceMappingURL=Select.d.ts.map