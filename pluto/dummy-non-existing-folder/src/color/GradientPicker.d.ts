import "./GradientPicker.css";
import { type Input } from "@synnaxlabs/lyra/input";
import { color, scale } from "@synnaxlabs/x";
import { type ReactElement } from "react";
interface GradientProps extends Input.Control<color.Stop[]> {
    scale?: scale.Scale<number>;
}
export declare const GradientPicker: ({ value, onChange, scale: scl, }: GradientProps) => ReactElement;
export {};
//# sourceMappingURL=GradientPicker.d.ts.map