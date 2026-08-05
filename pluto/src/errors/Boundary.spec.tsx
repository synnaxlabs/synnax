// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Errors } from "@/errors";

// Boundary renders Fallback on error, which kicks off a source-map resolution effect.
// Stub stacktrace-js at the library boundary so that effect doesn't hit the network.
vi.mock("stacktrace-js", () => ({
  default: {
    fromError: async () => {
      throw new Error("no maps in test env");
    },
  },
}));

describe("Boundary", () => {
  let consoleSpy: ReturnType<typeof vi.spyOn>;
  let warnSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    // React logs caught errors via console.error; Fallback warns when source-map
    // resolution fails. Silence both so the test output stays clean.
    consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
  });

  afterEach(() => {
    consoleSpy.mockRestore();
    warnSpy.mockRestore();
  });

  interface ThrowingComponentProps {
    shouldThrow: boolean;
  }

  const ThrowingComponent = ({ shouldThrow }: ThrowingComponentProps) => {
    if (shouldThrow) throw new Error("Test error message");
    return <div>Content rendered successfully</div>;
  };

  it("should render children when no error occurs", () => {
    const c = render(
      <Errors.Boundary>
        <ThrowingComponent shouldThrow={false} />
      </Errors.Boundary>,
    );
    expect(c.getByText("Content rendered successfully")).toBeTruthy();
  });

  it("should render fallback when an error occurs", () => {
    const c = render(
      <Errors.Boundary>
        <ThrowingComponent shouldThrow />
      </Errors.Boundary>,
    );
    expect(c.getByText("Test error message")).toBeTruthy();
  });

  it("should call onError when an error occurs", () => {
    const onError = vi.fn();
    render(
      <Errors.Boundary onError={onError}>
        <ThrowingComponent shouldThrow />
      </Errors.Boundary>,
    );
    expect(onError).toHaveBeenCalled();
    expect(onError.mock.calls[0][0].message).toBe("Test error message");
  });

  it("should use custom FallbackComponent when provided", () => {
    const CustomFallback = () => <div>Custom fallback</div>;
    const c = render(
      <Errors.Boundary FallbackComponent={CustomFallback}>
        <ThrowingComponent shouldThrow />
      </Errors.Boundary>,
    );
    expect(c.getByText("Custom fallback")).toBeTruthy();
  });

  it("should pass componentStack to FallbackComponent", () => {
    const CustomFallback = ({ componentStack }: Errors.FallbackProps) => (
      <div data-testid="stack">{componentStack}</div>
    );
    const c = render(
      <Errors.Boundary FallbackComponent={CustomFallback}>
        <ThrowingComponent shouldThrow />
      </Errors.Boundary>,
    );
    const stack = c.getByTestId("stack");
    expect(stack.textContent).toContain("ThrowingComponent");
  });

  it("should reset when resetErrorBoundary is called", () => {
    let shouldThrow = true;
    const TestComponent = () => {
      if (shouldThrow) throw new Error("Test error");
      return <div>Recovered</div>;
    };
    const c = render(
      <Errors.Boundary>
        <TestComponent />
      </Errors.Boundary>,
    );
    expect(c.getByText("Test error")).toBeTruthy();
    shouldThrow = false;
    fireEvent.click(c.getByText("Reload"));
    expect(c.getByText("Recovered")).toBeTruthy();
  });

  describe("ResetProvider", () => {
    it("should reset when the surrounding reset value changes", () => {
      let shouldThrow = true;
      const TestComponent = () => {
        if (shouldThrow) throw new Error("Test error");
        return <div>Recovered</div>;
      };
      const renderAt = (value: number) => (
        <Errors.ResetProvider value={value}>
          <Errors.Boundary>
            <TestComponent />
          </Errors.Boundary>
        </Errors.ResetProvider>
      );
      const c = render(renderAt(0));
      expect(c.getByText("Test error")).toBeTruthy();
      shouldThrow = false;
      c.rerender(renderAt(1));
      expect(c.getByText("Recovered")).toBeTruthy();
    });

    it("should re-catch an error that outlives the reset", () => {
      const TestComponent = () => {
        throw new Error("Test error");
      };
      const renderAt = (value: number) => (
        <Errors.ResetProvider value={value}>
          <Errors.Boundary>
            <TestComponent />
          </Errors.Boundary>
        </Errors.ResetProvider>
      );
      const c = render(renderAt(0));
      c.rerender(renderAt(1));
      expect(c.getByText("Test error")).toBeTruthy();
    });

    it("should not reset a boundary that never caught", () => {
      const onReset = vi.fn();
      const renderAt = (value: number) => (
        <Errors.ResetProvider value={value}>
          <Errors.Boundary onReset={onReset}>
            <ThrowingComponent shouldThrow={false} />
          </Errors.Boundary>
        </Errors.ResetProvider>
      );
      const c = render(renderAt(0));
      c.rerender(renderAt(1));
      expect(onReset).not.toHaveBeenCalled();
      expect(c.getByText("Content rendered successfully")).toBeTruthy();
    });
  });
});
