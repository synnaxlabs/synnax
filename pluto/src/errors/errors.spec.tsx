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

interface ThrowingComponentProps {
  shouldThrow: boolean;
}

const ThrowingComponent = ({ shouldThrow }: ThrowingComponentProps) => {
  if (shouldThrow) throw new Error("Test error message");
  return <div>Content rendered successfully</div>;
};

describe("Error", () => {
  let consoleSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    consoleSpy.mockRestore();
  });

  describe("Boundary", () => {
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
  });

  describe("Fallback", () => {
    const mockError = new Error("Something went wrong");
    mockError.name = "TestError";
    const mockReset = vi.fn();

    it("should render the error name", () => {
      const c = render(
        <Errors.Fallback error={mockError} resetErrorBoundary={mockReset} />,
      );
      expect(c.getByText("TestError")).toBeTruthy();
    });

    it("should render the error message", () => {
      const c = render(
        <Errors.Fallback error={mockError} resetErrorBoundary={mockReset} />,
      );
      expect(c.getByText("Something went wrong")).toBeTruthy();
    });

    it("should render the default reload button", () => {
      const c = render(
        <Errors.Fallback error={mockError} resetErrorBoundary={mockReset} />,
      );
      expect(c.getByText("Reload")).toBeTruthy();
    });

    it("should call resetErrorBoundary when reload button is clicked", () => {
      const resetFn = vi.fn();
      const c = render(
        <Errors.Fallback error={mockError} resetErrorBoundary={resetFn} />,
      );
      fireEvent.click(c.getByText("Reload"));
      expect(resetFn).toHaveBeenCalled();
    });

    it("should render custom children instead of default button", () => {
      const customAction = vi.fn();
      const c = render(
        <Errors.Fallback error={mockError} resetErrorBoundary={mockReset}>
          <button onClick={customAction}>Custom Action</button>
        </Errors.Fallback>,
      );
      expect(c.queryByText("Reload")).not.toBeTruthy();
      expect(c.getByText("Custom Action")).toBeTruthy();
      fireEvent.click(c.getByText("Custom Action"));
      expect(customAction).toHaveBeenCalled();
    });

    it("should render the logo when showLogo is true", () => {
      const c = render(
        <Errors.Fallback error={mockError} resetErrorBoundary={mockReset} showLogo />,
      );
      expect(c.container.querySelector("svg")).toBeTruthy();
    });

    it("should not render the logo by default", () => {
      const c = render(
        <Errors.Fallback error={mockError} resetErrorBoundary={mockReset} />,
      );
      expect(c.container.querySelector("svg")).not.toBeTruthy();
    });

    describe("compact variant", () => {
      it("should use smaller text sizes", () => {
        const c = render(
          <Errors.Fallback
            error={mockError}
            resetErrorBoundary={mockReset}
            variant="compact"
          />,
        );
        const errorName = c.getByText("TestError");
        expect(errorName.className).toContain("h3");
      });
    });

    describe("full variant", () => {
      it("should use larger text sizes", () => {
        const c = render(
          <Errors.Fallback
            error={mockError}
            resetErrorBoundary={mockReset}
            variant="full"
          />,
        );
        const errorName = c.getByText("TestError");
        expect(errorName.className).toContain("h1");
      });
    });

    describe("error stack", () => {
      it("should render the error stack when present", () => {
        const errorWithStack = new Error("Test");
        errorWithStack.stack = "Error: Test\n    at TestComponent";
        const c = render(
          <Errors.Fallback error={errorWithStack} resetErrorBoundary={mockReset} />,
        );
        expect(c.getByText(/at TestComponent/)).toBeTruthy();
      });

      it("should not render stack section when stack is empty", () => {
        const errorNoStack = new Error("Test");
        errorNoStack.stack = "";
        const c = render(
          <Errors.Fallback error={errorNoStack} resetErrorBoundary={mockReset} />,
        );
        expect(c.container.querySelector(".pluto-error-fallback__stack")).toBeFalsy();
      });
    });
  });
});
