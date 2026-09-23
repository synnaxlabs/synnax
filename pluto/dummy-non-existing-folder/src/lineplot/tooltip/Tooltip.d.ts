import { type ReactElement } from "react";
import { type z } from "zod";
import { Aether } from "../../aether";
import { tooltip } from "./aether";
export interface TooltipProps extends Omit<z.input<typeof tooltip.tooltipStateZ>, "position">, Aether.ComponentProps {
}
export declare const Tooltip: ({ aetherKey, ...rest }: TooltipProps) => ReactElement | null;
//# sourceMappingURL=Tooltip.d.ts.map