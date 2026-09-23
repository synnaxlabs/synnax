import { type channel } from "@synnaxlabs/client";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";
import { type ListQuery } from "./queries";
import { type Flux } from "../flux";
export interface SelectMultipleProps extends Omit<Select.MultipleProps<channel.Key, channel.Channel | undefined>, "resourceName" | "data" | "getItem" | "subscribe" | "children">, Flux.UseListParams<ListQuery, channel.Key, channel.Channel> {
}
export declare const SelectMultiple: ({ initialQuery, filter, ...rest }: SelectMultipleProps) => ReactElement;
export interface SelectSingleProps extends Omit<Select.SingleProps<channel.Key, channel.Channel | undefined>, "data" | "getItem" | "subscribe" | "children" | "resourceName">, Flux.UseListParams<ListQuery, channel.Key, channel.Channel> {
}
export declare const SelectSingle: ({ initialQuery, filter, ...rest }: SelectSingleProps) => ReactElement;
//# sourceMappingURL=Select.d.ts.map