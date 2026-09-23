import { type ReactElement } from "react";
import { Flex } from "../flex";
export interface BackgroundProps extends Flex.BoxProps {
    visible: boolean;
}
export declare const BACKGROUND_CLASS: string;
/** The backdrop behind a modal {@link Dialog}. It closes the dialog when clicked. */
export declare const Background: ({ children, visible, ...rest }: BackgroundProps) => ReactElement;
//# sourceMappingURL=Background.d.ts.map