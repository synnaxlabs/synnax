import { type ranger } from "@synnaxlabs/client";
import { Select as Base } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";
import { type Flux } from "../flux";
import { type ListQuery } from "./queries";
export interface SelectProps extends Omit<Base.SingleProps<ranger.Key, ranger.Payload | undefined>, "data" | "getItem" | "subscribe" | "status" | "onFetchMore" | "onSearch" | "children" | "resourceName">, Flux.UseListParams<ListQuery, ranger.Key, ranger.Payload> {
}
export declare const Select: ({ filter, initialQuery, ...rest }: SelectProps) => ReactElement;
//# sourceMappingURL=Select.d.ts.map