import { type ReactElement } from "react";
export interface CellProps extends React.ComponentPropsWithRef<"td"> {
    selected?: boolean;
    height: number;
}
export declare const Cell: ({ ref, children, className, selected, height, style, ...rest }: CellProps) => ReactElement;
//# sourceMappingURL=Cell.d.ts.map