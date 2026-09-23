import "./Swatch.css";
import { Button } from "@synnaxlabs/lyra/button";
import { Haul } from "@synnaxlabs/lyra/haul";
import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";
export declare const HAUL_TYPE = "color";
export type HaulItem = Haul.Item<typeof HAUL_TYPE, color.Hex, undefined>;
export declare const createHaulItem: (key: color.Hex) => HaulItem;
export declare const isHaulItem: (item: Haul.Item) => item is HaulItem;
export declare const filterHaulItems: (items: Haul.Item[]) => HaulItem[];
export declare const canDropHaulItem: Haul.CanDrop;
export interface BaseSwatchProps extends Omit<Button.ButtonProps, "onChange" | "value" | "size"> {
    value: color.Crude;
    onChange?: (c: color.Color) => void;
    size?: Button.ButtonProps["size"] | "tiny";
}
export declare const BaseSwatch: ({ value, onChange, className, size, draggable, style, ...rest }: BaseSwatchProps) => ReactElement;
//# sourceMappingURL=BaseSwatch.d.ts.map