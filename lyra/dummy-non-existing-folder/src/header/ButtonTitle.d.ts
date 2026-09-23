import { type ReactElement } from "react";
import { Button } from "../button";
/** Props for {@link ButtonTitle}. */
export interface ButtonTitleProps extends Omit<Button.ButtonProps, "variant" | "size"> {
}
/** A {@link Title} the user can click. It takes its size from the enclosing header. */
export declare const ButtonTitle: ({ children, className, onClick, ...rest }: ButtonTitleProps) => ReactElement;
//# sourceMappingURL=ButtonTitle.d.ts.map