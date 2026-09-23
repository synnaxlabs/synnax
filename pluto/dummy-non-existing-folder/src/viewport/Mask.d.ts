import "./Mask.css";
import { type ReactElement } from "react";
import { type UseReturn } from "./use";
type DivProps = React.DetailedHTMLProps<React.HTMLAttributes<HTMLDivElement>, HTMLDivElement>;
export interface MaskProps extends Omit<UseReturn, "ref">, Omit<DivProps, "onDragStart" | "onDragEnd" | "onDrag" | "ref" | "onDoubleClick"> {
}
export declare const Mask: ({ className, mode, maskBox, children, style, ...rest }: MaskProps) => ReactElement | null;
export {};
//# sourceMappingURL=Mask.d.ts.map