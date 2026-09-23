import "./Dialog.css";
import { Flex } from "../flex";
/** Props for {@link Dialog}. */
export interface DialogProps extends Flex.BoxProps<"div"> {
    /** Keeps the children mounted and hidden while closed, instead of unmounting them. */
    passthrough?: boolean;
}
/**
 * The floating surface of a {@link Frame}. It mounts only while open, unless
 * `passthrough` is set, and portals itself to the document root when modal.
 */
export declare const Dialog: ({ style, background, className, bordered, rounded, passthrough, children, ...rest }: DialogProps) => import("react").JSX.Element | null;
//# sourceMappingURL=Dialog.d.ts.map