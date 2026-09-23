import "./Loading.css";
import { type ReactElement } from "react";
import { Flex } from "../flex";
export interface LoadingProps extends Flex.BoxProps {
}
/**
 * Loading centers an indicator in its container and holds it invisible until the
 * wait is long enough to be worth reporting, so a fast read never flashes one.
 * Defaults to the inline glyph; pass {@link Orbital} for a surface with room for
 * it. Inline surfaces (a tab chip, an icon slot) should render
 * {@link Icon.Loading} directly instead of wrapping it here.
 */
export declare const Loading: ({ className, children, ...rest }: LoadingProps) => ReactElement;
//# sourceMappingURL=Loading.d.ts.map