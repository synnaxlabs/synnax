import "./Item.css";
import { type record } from "@synnaxlabs/x";
import { type Button } from "../button";
import { Select } from "../select";
export type ItemProps<K extends record.Key, E extends Button.ElementType = "div"> = Select.ListItemProps<K, E> & {
    loading?: boolean;
    useMargin?: boolean;
    offsetMultiplier?: number;
};
export declare const Item: <K extends record.Key, E extends Button.ElementType = "div">({ children, style, className, loading, useMargin, offsetMultiplier, ...rest }: ItemProps<K, E>) => import("react").JSX.Element;
//# sourceMappingURL=Item.d.ts.map