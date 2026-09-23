import "./Select.css";
import { Select as BaseSelect } from "@synnaxlabs/lyra/select";
import { notation } from "@synnaxlabs/x";
import { type ReactElement } from "react";
export interface SelectNotationProps extends Omit<BaseSelect.ButtonsProps<notation.Notation>, "keys"> {
}
export declare const Select: ({ className, ...rest }: SelectNotationProps) => ReactElement;
//# sourceMappingURL=Select.d.ts.map