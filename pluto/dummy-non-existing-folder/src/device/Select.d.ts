import "./Select.css";
import { type device } from "@synnaxlabs/client";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";
import { type ListParams } from "./queries";
import { type Flux } from "../flux";
export interface SelectSingleProps extends Omit<Select.SingleProps<device.Key, device.Device | undefined>, "resourceName" | "data" | "getItem" | "subscribe" | "children">, Flux.UseListParams<ListParams, device.Key, device.Device> {
}
export declare const SelectSingle: ({ filter, initialQuery, ...rest }: SelectSingleProps) => ReactElement;
//# sourceMappingURL=Select.d.ts.map