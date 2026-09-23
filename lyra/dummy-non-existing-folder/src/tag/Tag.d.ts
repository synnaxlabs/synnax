import "./Tag.css";
import { color, type optional } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { Button } from "../button";
import { type Component } from "../component";
import { Icon } from "../icon";
export interface TagProps extends optional.Optional<Omit<Button.ButtonProps<"div">, "size" | "wrap" | "color">, "level"> {
    icon?: Icon.ReactElement;
    onClose?: () => void;
    color?: color.Crude;
    size?: Component.Size;
    variant?: "filled" | "outlined";
}
export declare const Tag: ({ children, size, color: pColor, icon, onClose, className, onDragStart, ...rest }: TagProps) => ReactElement;
//# sourceMappingURL=Tag.d.ts.map