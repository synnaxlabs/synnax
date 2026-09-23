import "./Rule.css";
import { Flex } from "@synnaxlabs/lyra/flex";
import { type ReactElement } from "react";
import { type z } from "zod";
import { Aether } from "../../aether";
import { rule } from "./aether";
export interface RuleProps extends Omit<z.input<typeof rule.ruleStateZ>, "dragging" | "pixelPosition">, Omit<Flex.BoxProps, "color">, Aether.ComponentProps {
    label?: string;
    onLabelChange?: (label: string) => void;
    units?: string;
    onUnitsChange?: (label: string) => void;
    onPositionChange?: (position: number) => void;
    onSelect?: () => void;
}
export declare const Rule: ({ aetherKey, label, position: propsPosition, onLabelChange, onPositionChange, onUnitsChange, units, color: colorVal, lineWidth, lineDash, className, onSelect, style, ...rest }: RuleProps) => ReactElement | null;
//# sourceMappingURL=Rule.d.ts.map