import "./Base.css";
import { type Icon } from "@synnaxlabs/lyra/icon";
import { type CSSProperties } from "react";
export interface HandleSpec {
    key: string;
    Icon: Icon.FC;
}
export interface BaseProps extends MinimalProps {
    type?: string;
    Icon?: Icon.ReactElement;
    color: string;
    textColor: string;
}
export interface TypeTextProps {
    type: string;
    icon: Icon.ReactElement;
    color: string;
    textColor: string;
}
export declare const TypeText: ({ type, icon, color, textColor }: TypeTextProps) => import("react").JSX.Element;
export declare const Base: ({ type, Icon: icon, sources, sinks, color, textColor, children, scale, }: BaseProps) => import("react").JSX.Element;
interface MinimalProps {
    sources?: HandleSpec[];
    centerSources?: boolean;
    sinks?: HandleSpec[];
    centerSinks?: boolean;
    children: React.ReactNode;
    style?: CSSProperties;
    scale?: number;
}
export declare const Minimal: ({ sources, sinks, children, style, centerSources, centerSinks, }: MinimalProps) => import("react").JSX.Element;
export {};
//# sourceMappingURL=Base.d.ts.map