import { HTMLAttributes, PropsWithChildren } from "react";
export interface SpaceProps extends PropsWithChildren<HTMLAttributes<HTMLDivElement>> {
    empty?: boolean;
    size?: "small" | "medium" | "large" | number;
    direction?: "horizontal" | "vertical";
    justify?: "start" | "end" | "center" | "spaceBetween" | "spaceAround" | "spaceEvenly";
    align?: "start" | "end" | "center" | "stretch";
}
declare const Space: ({ empty, size, justify, children, align, ...props }: SpaceProps) => JSX.Element;
export default Space;
