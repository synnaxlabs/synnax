import { type access } from "@synnaxlabs/client";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";
import { type ListParams } from "./queries";
import { type Flux } from "../../flux";
export interface SelectMultipleProps extends Omit<Select.MultipleProps<access.policy.Key, access.policy.Policy | undefined>, "resourceName" | "data" | "getItem" | "subscribe" | "children">, Flux.UseListParams<ListParams, access.policy.Key, access.policy.Policy> {
}
export declare const SelectMultiple: ({ initialQuery, filter, ...rest }: SelectMultipleProps) => ReactElement;
export interface SelectSingleProps extends Omit<Select.SingleProps<access.policy.Key, access.policy.Policy | undefined>, "data" | "getItem" | "subscribe" | "children" | "resourceName">, Flux.UseListParams<ListParams, access.policy.Key, access.policy.Policy> {
}
export declare const SelectSingle: ({ initialQuery, filter, ...rest }: SelectSingleProps) => ReactElement;
//# sourceMappingURL=Select.d.ts.map