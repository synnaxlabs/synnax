import { type ReactElement, type ReactNode } from "react";
import { Flex } from "../flex";
/** Props for {@link Actions}. */
export interface ActionsProps extends Omit<Flex.BoxProps, "children" | "direction"> {
    children?: ReactNode;
}
/** The trailing slot of a {@link Header}, holding its buttons. */
export declare const Actions: ({ children, ...rest }: ActionsProps) => ReactElement;
//# sourceMappingURL=Actions.d.ts.map