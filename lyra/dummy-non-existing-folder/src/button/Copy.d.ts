import { type CrudeTimeSpan } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { type ButtonProps } from "./Button";
/** Props for {@link Copy}. */
export interface CopyProps extends ButtonProps {
    /** The text to copy, or a function returning it. The function may be async. */
    text: string | (() => string | Promise<string>);
    /** Called once the text reaches the clipboard. */
    onCopy?: () => void;
    /** How long the checkmark stays up. Defaults to 2 seconds. */
    copiedDuration?: CrudeTimeSpan;
    /** The status shown on success. Omit for no status. */
    successMessage?: string | (() => string);
}
/**
 * A button that copies text to the clipboard and shows a checkmark on success.
 *
 * @example
 * <Button.Copy text="Hello, world!" tooltip="Copy greeting" />
 * @example
 * <Button.Copy
 *   text={JSON.stringify(data)}
 *   onCopy={() => console.log("Copied!")}
 *   variant="filled"
 * />
 */
export declare const Copy: ({ text, onCopy, copiedDuration, successMessage, tooltip, children, onClick, ...rest }: CopyProps) => ReactElement;
//# sourceMappingURL=Copy.d.ts.map