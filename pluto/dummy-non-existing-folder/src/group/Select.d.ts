import { type group } from "@synnaxlabs/client";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";
import { type Flux } from "../flux";
import { type ListQuery } from "./queries";
export interface SelectSingleProps extends Omit<Select.SingleProps<group.Key, group.Group | undefined>, "data" | "resourceName" | "subscribe" | "children">, Flux.UseListParams<ListQuery, group.Key, group.Group> {
}
export declare const SelectSingle: ({ filter, initialQuery, ...rest }: SelectSingleProps) => ReactElement;
//# sourceMappingURL=Select.d.ts.map