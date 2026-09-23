import { type status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type CSSProperties, type ReactElement } from "react";
import { type z } from "zod";
import { control } from "./aether";
export interface ChipProps extends Pick<z.input<typeof control.chipStateZ>, "source" | "sink">, Omit<Button.ButtonProps, "onClick" | "children"> {
}
interface ChipStyle {
    message: string;
    chipColor: string;
    chipIcon: Icon.FC;
    buttonStyle?: CSSProperties;
    disabled?: boolean;
}
export declare const tooltipMessage: (status: status.Status<typeof control.chipStatusDetailsZ>) => ChipStyle;
export declare const Chip: ({ source, sink, className, ...rest }: ChipProps) => ReactElement;
export {};
//# sourceMappingURL=Chip.d.ts.map