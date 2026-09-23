import "./Select.css";
import { type access } from "@synnaxlabs/client";
import { Select as Base } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";
import { type ListQuery } from "./queries";
import { type Flux } from "../../flux";
export interface SelectProps extends Omit<Base.SingleProps<access.role.Key, access.role.Role | undefined>, "resourceName" | "data" | "getItem" | "subscribe" | "children">, Flux.UseListParams<ListQuery, access.role.Key, access.role.Role> {
}
export declare const Select: ({ initialQuery, filter, ...props }: SelectProps) => ReactElement;
//# sourceMappingURL=Select.d.ts.map