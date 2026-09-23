import "./Bar.css";
import { location } from "@synnaxlabs/x";
import { type FunctionComponent, type ReactElement } from "react";
import { Flex } from "../flex";
export interface BarProps extends Omit<Flex.BoxProps, "direction" | "size" | "ref"> {
    location?: location.Crude;
    size?: string | number;
    bordered?: boolean;
}
declare const BaseBar: ({ location: propsLoc, size, className, style: propsStyle, bordered, ...rest }: BarProps) => ReactElement;
export interface BarContentProps extends Omit<Flex.BoxProps<"div">, "ref"> {
}
type BaseBarType = typeof BaseBar;
declare const Start: FunctionComponent<BarContentProps>;
declare const End: FunctionComponent<BarContentProps>;
declare const Center: FunctionComponent<BarContentProps>;
declare const Content: FunctionComponent<BarContentProps>;
declare const AbsoluteCenter: FunctionComponent<BarContentProps>;
export interface BarType extends BaseBarType {
    Start: typeof Start;
    Center: typeof Center;
    End: typeof End;
    AbsoluteCenter: typeof AbsoluteCenter;
    Content: typeof Content;
}
export declare const Bar: BarType;
export {};
//# sourceMappingURL=Bar.d.ts.map