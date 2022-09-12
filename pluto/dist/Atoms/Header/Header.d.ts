import { SpaceProps } from "../Space/Space";
import { ReactElement } from "react";
export declare type FontSize = "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";
export interface HeadingProps extends Omit<SpaceProps, "children" | "size"> {
    size: FontSize;
    text: string;
    icon?: ReactElement;
}
declare const Header: ({ size, text, icon, style, className, ...props }: HeadingProps) => JSX.Element;
export default Header;
