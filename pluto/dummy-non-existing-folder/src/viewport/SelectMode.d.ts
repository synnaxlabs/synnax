import { Select } from "@synnaxlabs/lyra/select";
import { type Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { type ReactElement } from "react";
import { type Mode, type UseTriggers } from "./use";
export type FilteredMode = Exclude<Mode, "cancel">;
interface TooltipProps {
    mode: FilteredMode;
    triggers: Triggers.Trigger[];
}
export declare const TooltipText: ({ mode, triggers }: TooltipProps) => ReactElement;
export interface SelectModeProps extends Omit<Select.ButtonsProps<Mode>, "keys">, Omit<Tooltip.ExtensionProps, "tooltip"> {
    triggers: UseTriggers;
    disable?: Mode[];
}
export declare const SelectMode: ({ triggers, value, onChange, disable, tooltipLocation, hideTooltip, ...rest }: SelectModeProps) => ReactElement;
export {};
//# sourceMappingURL=SelectMode.d.ts.map