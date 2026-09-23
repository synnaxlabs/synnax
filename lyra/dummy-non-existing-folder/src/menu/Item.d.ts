import "./Item.css";
import { type ReactElement } from "react";
import { Button } from "../button";
export interface ItemProps extends Button.ButtonProps {
    itemKey: string;
}
export declare const Item: ({ itemKey, className, onClick, size, ...rest }: ItemProps) => ReactElement;
//# sourceMappingURL=Item.d.ts.map