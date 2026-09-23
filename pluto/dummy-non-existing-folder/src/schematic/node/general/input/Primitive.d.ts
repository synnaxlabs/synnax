import "./input.css";
import { type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";
interface PrimitiveProps extends Partial<Omit<schematic.InputNodeConfig, "variant" | "label" | "scale">> {
    initialValue?: string;
    className?: string;
    onSend?: (value: string) => void;
}
export declare const Input: ({ className, initialValue, orientation, color, size, onSend, disabled, onClickDelay, }: PrimitiveProps) => ReactElement;
export {};
//# sourceMappingURL=Primitive.d.ts.map