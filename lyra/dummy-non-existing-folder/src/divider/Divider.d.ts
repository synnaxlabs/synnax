import "./Divider.css";
import { type location } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { Flex } from "../flex";
/** The props for the {@link Divider} component. */
export interface DividerProps extends Flex.BoxProps {
    padded?: boolean | location.Location;
}
/** Divider renders a vertical or horizontal divided to separate content. */
export declare const Divider: ({ className, padded, color, ...rest }: DividerProps) => ReactElement;
//# sourceMappingURL=Divider.d.ts.map