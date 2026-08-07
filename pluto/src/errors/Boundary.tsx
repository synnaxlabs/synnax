// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Component,
  type ComponentType,
  type ErrorInfo,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
} from "react";

import { context } from "@/context";
import { Fallback, type FallbackProps } from "@/errors/Fallback";

const [ResetContext] = context.create({ defaultValue: 0, displayName: "Errors.Reset" });

export interface ResetProviderProps extends PropsWithChildren {
  value: number;
}

/**
 * Clears the error on every {@link Boundary} below whenever `value` changes. Drive it
 * from a signal meaning "the world may have changed" (a reconnect, a new cluster) so
 * boundaries that latched on a transient failure try again.
 */
export const ResetProvider = ({
  value,
  children,
}: ResetProviderProps): ReactElement => (
  <ResetContext value={value}>{children}</ResetContext>
);

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
export class Boundary extends Component<BoundaryProps, BoundaryState> {
  static contextType = ResetContext;
  declare context: number;

  state: BoundaryState = {
    error: null,
    componentStack: null,
    caughtAt: null,
  };

  static getDerivedStateFromError(error: Error): Partial<BoundaryState> {
    return { error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    this.setState({
      componentStack: errorInfo.componentStack ?? null,
      caughtAt: this.context,
    });
    this.props.onError?.(error, errorInfo);
  }

  componentDidUpdate(): void {
    const { error, caughtAt } = this.state;
    if (error != null && caughtAt != null && caughtAt !== this.context)
      this.resetErrorBoundary();
  }

  resetErrorBoundary = (): void => {
    if (this.state.error == null) return;
    this.props.onReset?.();
    this.setState({ error: null, componentStack: null, caughtAt: null });
  };

  render(): ReactNode {
    const { error, componentStack } = this.state;
    const { children, FallbackComponent = Fallback } = this.props;

    if (error != null)
      return (
        <FallbackComponent
          error={error}
          componentStack={componentStack}
          resetErrorBoundary={this.resetErrorBoundary}
        />
      );

    return children;
  }
}
