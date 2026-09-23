import { Component, type ComponentType, type ErrorInfo, type PropsWithChildren, type ReactElement, type ReactNode } from "react";
import { type FallbackProps } from "./Fallback";
export interface ResetProviderProps extends PropsWithChildren {
    value: number;
}
/**
 * Clears the error on every {@link Boundary} below whenever `value` changes. Drive it
 * from a signal meaning "the world may have changed" (a reconnect, a new cluster) so
 * boundaries that latched on a transient failure try again.
 */
export declare const ResetProvider: ({ value, children, }: ResetProviderProps) => ReactElement;
export interface BoundaryProps {
    /** The children to render. */
    children?: ReactNode;
    /** Custom fallback component to render when an error occurs. */
    FallbackComponent?: ComponentType<FallbackProps>;
    /** Callback invoked when an error is caught. */
    onError?: (error: Error, info: ErrorInfo) => void;
    /** Callback invoked when the error boundary resets. */
    onReset?: () => void;
}
interface BoundaryState {
    error: Error | null;
    componentStack: string | null;
    /** Reset value current when the error was caught; null while there is no error. */
    caughtAt: number | null;
}
/**
 * Error boundary component that catches errors in its children and displays a fallback
 * UI. Implemented as a React class component to use componentDidCatch lifecycle.
 * Resets itself when the surrounding {@link ResetProvider} value changes.
 */
export declare class Boundary extends Component<BoundaryProps, BoundaryState> {
    static contextType: import("react").Context<number>;
    context: number;
    state: BoundaryState;
    static getDerivedStateFromError(error: Error): Partial<BoundaryState>;
    componentDidCatch(error: Error, errorInfo: ErrorInfo): void;
    componentDidUpdate(): void;
    resetErrorBoundary: () => void;
    render(): ReactNode;
}
export {};
//# sourceMappingURL=Boundary.d.ts.map