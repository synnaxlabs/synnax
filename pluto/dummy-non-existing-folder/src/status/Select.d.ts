import { type status } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { List } from "@synnaxlabs/lyra/list";
import { Select as Base } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";
import { type Flux } from "../flux";
import { type ListParams } from "./queries";
export interface SelectProps extends Omit<Base.SingleProps<status.Key, status.Status>, "data" | "getItem" | "subscribe" | "status" | "onFetchMore" | "onSearch" | "children" | "resourceName">, Flux.UseListParams<ListParams, status.Key, status.Status> {
}
export declare const Select: ({ initialQuery, filter, ...props }: SelectProps) => ReactElement;
export declare const listItemRenderProp: Component.RenderProp<List.ItemProps<string, "div">, ReactElement<unknown, string | import("react").JSXElementConstructor<any>> | null>;
//# sourceMappingURL=Select.d.ts.map