import { Select as BaseSelect } from "@synnaxlabs/lyra/select";
import { direction } from "@synnaxlabs/x";
import { type ReactElement } from "react";
export interface SelectProps extends Omit<BaseSelect.ButtonsProps<direction.Direction>, "keys"> {
    yDirection?: "up" | "down";
}
export declare const Select: ({ yDirection, ...rest }: SelectProps) => ReactElement;
//# sourceMappingURL=Select.d.ts.map