import { type ReactElement } from "react";
import { type Size } from "./size";
import { type Select } from "../select";
/** Props for {@link SelectSize}. */
export interface SelectComponentSizeProps extends Omit<Select.ButtonsProps<Size>, "keys"> {
}
/** A button group for picking a {@link Size}, labeled XS through XL. */
export declare const SelectSize: (props: SelectComponentSizeProps) => ReactElement;
//# sourceMappingURL=SelectSize.d.ts.map