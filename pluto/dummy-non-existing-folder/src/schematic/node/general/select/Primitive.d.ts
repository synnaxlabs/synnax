import "./select.css";
import { type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";
interface RenderProps extends Partial<Pick<schematic.SelectNodeConfig, "color" | "orientation" | "size" | "disabled" | "inlineSize" | "onClickDelay">> {
    options: schematic.SelectNodeConfig["options"];
    className?: string;
    value?: string;
    onChange: (key: string | null) => void;
    onSend?: (value: number) => void;
}
export declare const Select: ({ className, orientation, color, value, onChange, onSend, options, size, disabled, inlineSize, onClickDelay, }: RenderProps) => ReactElement;
export {};
//# sourceMappingURL=Primitive.d.ts.map