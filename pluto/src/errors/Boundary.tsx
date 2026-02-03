// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component, type ComponentType, type ErrorInfo, type ReactNode } from "react";

import { Fallback, type FallbackProps } from "@/errors/Fallback";

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
}

/**
 * Error boundary component that catches errors in its children and displays a fallback
 * UI. Implemented as a React class component to use componentDidCatch lifecycle.
 */
export class Boundary extends Component<BoundaryProps, BoundaryState> {
  state: BoundaryState = {
    error: null,
    componentStack: null,
  };

  static getDerivedStateFromError(error: Error): Partial<BoundaryState> {
    return { error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    this.setState({ componentStack: errorInfo.componentStack ?? null });
    this.props.onError?.(error, errorInfo);
  }

  resetErrorBoundary = (): void => {
    this.props.onReset?.();
    this.setState({ error: null, componentStack: null });
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
