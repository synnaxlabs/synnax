import "./Fallback.css";
import { type record } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";
/** Props for the error fallback component. */
export interface FallbackProps extends PropsWithChildren {
    /** The error that was caught. */
    error: Error;
    /** The React component stack trace from the error boundary. */
    componentStack?: string | null;
    /** Function to reset the error boundary and retry rendering. */
    resetErrorBoundary: () => void;
    /** Whether to show the Synnax logo above the error details. Defaults to false. */
    showLogo?: boolean;
    /** Extra information to copying to the clipboard when the user clicks the "Copy"
     * button. */
    extraInfo?: record.Unknown;
}
/**
 * Default error fallback component. Can be used standalone or with ErrorBoundary.
 * Supports both compact (for mosaic leafs) and full (for page overlays) variants.
 *
 * @example
 * // With default retry button
 * <Fallback error={error} resetErrorBoundary={reset} />
 * @example
 * // With custom actions
 * <Fallback error={error} resetErrorBoundary={reset} icon={<Logo />}>
 *   <Button onClick={reset}>Try again</Button>
 *   <Button onClick={clear}>Clear storage</Button>
 * </Fallback>
 */
export declare const Fallback: ({ error, componentStack, resetErrorBoundary, children, extraInfo, }: FallbackProps) => ReactElement;
//# sourceMappingURL=Fallback.d.ts.map