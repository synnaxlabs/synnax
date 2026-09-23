import { type schematic } from "@synnaxlabs/client";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";
import { type Flux } from "../../flux";
import { type ListQuery } from "./queries";
export interface SelectSingleProps extends Omit<Select.SingleProps<schematic.symbol.Key, schematic.symbol.Symbol | undefined>, "data" | "resourceName" | "subscribe" | "children">, Flux.UseListParams<ListQuery, schematic.symbol.Key, schematic.symbol.Symbol> {
}
export declare const SelectSingle: ({ filter, initialQuery, ...rest }: SelectSingleProps) => ReactElement;
//# sourceMappingURL=Select.d.ts.map