import { type ReactElement } from "react";
import { type ButtonProps } from "./Button";
/** Props for {@link Close}. */
export interface CloseProps extends ButtonProps {
}
/**
 * A destructive glyph close that overlays its host's leading icon and reveals on
 * host hover. The host marks itself as a reveal container. Not a tab stop: the
 * keyboard path to closing belongs to the host (e.g. Delete on a focused tab).
 */
export declare const Close: ({ className, children, ...rest }: CloseProps) => ReactElement;
//# sourceMappingURL=Close.d.ts.map