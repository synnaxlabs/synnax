import "./textBox.css";
import { type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";
interface RenderProps extends Omit<schematic.TextBoxNodeConfig, "variant" | "label" | "scale"> {
    className?: string;
    onChange?: (value: string) => void;
}
export declare const TextBox: ({ className, orientation, width, color: colorVal, level, autoFitDisabled, align, value, onChange, }: RenderProps) => ReactElement;
export {};
//# sourceMappingURL=Primitive.d.ts.map