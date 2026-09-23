import { Select } from "@synnaxlabs/lyra/select";
import { Status as Base } from "@synnaxlabs/lyra/status";
import { type ReactElement } from "react";
type Entry = Select.StaticEntry<Base.Variant>;
export interface SelectMultipleVariantProps extends Omit<Select.MultipleProps<Base.Variant, Entry>, "data" | "getItem" | "subscribe" | "children" | "resourceName" | "onSearch"> {
}
export declare const SelectMultipleVariants: (props: SelectMultipleVariantProps) => ReactElement;
export {};
//# sourceMappingURL=SelectMultipleVariants.d.ts.map