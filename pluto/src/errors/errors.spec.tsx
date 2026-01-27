// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, waitFor } from "@testing-library/react";
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
  });

  describe("Fallback", () => {
    const mockError = new Error("Test error message");
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
      expect(c.getByText("Test error message")).toBeTruthy();
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

    it("should render the logo in the navbar", () => {
      const c = render(
        <Errors.Fallback error={mockError} resetErrorBoundary={mockReset} />,
      );
      expect(c.container.querySelector(".synnax-logo")).toBeTruthy();
    });

    describe("component stack", () => {
      it("should render the component stack when present", () => {
        const c = render(
          <Errors.Fallback
            error={mockError}
            componentStack={"    at ThrowingComponent\n    at Boundary"}
            resetErrorBoundary={mockReset}
          />,
        );
        expect(c.getByText(/at ThrowingComponent/)).toBeTruthy();
      });

      it("should fall back to error.stack when componentStack is null", () => {
        const c = render(
          <Errors.Fallback
            error={mockError}
            componentStack={null}
            resetErrorBoundary={mockReset}
          />,
        );
        expect(c.container.querySelector(".pluto-error-fallback__stack")).toBeTruthy();
      });

      it("should fall back to error.stack when componentStack is empty", () => {
        const c = render(
          <Errors.Fallback
            error={mockError}
            componentStack=""
            resetErrorBoundary={mockReset}
          />,
        );
        expect(c.container.querySelector(".pluto-error-fallback__stack")).toBeTruthy();
      });

      it("should not render stack section when both componentStack and error.stack are empty", () => {
        const errorWithoutStack = new Error("No stack");
        errorWithoutStack.stack = "";
        const c = render(
          <Errors.Fallback
            error={errorWithoutStack}
            componentStack=""
            resetErrorBoundary={mockReset}
          />,
        );
        expect(c.container.querySelector(".pluto-error-fallback__stack")).toBeFalsy();
      });
    });

    describe("copy diagnostics", () => {
      let writeTextMock: ReturnType<typeof vi.fn>;

      beforeEach(() => {
        writeTextMock = vi.fn().mockResolvedValue(undefined);
        Object.assign(navigator, {
          clipboard: { writeText: writeTextMock },
        });
      });

      it("should copy error name and message to clipboard", async () => {
        const c = render(
          <Errors.Fallback error={mockError} resetErrorBoundary={mockReset} />,
        );
        fireEvent.click(c.getByText("Copy diagnostics"));
        await waitFor(() => {
          expect(writeTextMock).toHaveBeenCalled();
        });
        const copiedText = writeTextMock.mock.calls[0][0];
        expect(copiedText).toContain("Error: TestError");
        expect(copiedText).toContain("Message: Test error message");
      });

      it("should include error stack trace in copied content", async () => {
        const errorWithStack = new Error("Stack test");
        errorWithStack.name = "StackError";
        errorWithStack.stack = "Error: Stack test\n    at TestFunction (test.js:1:1)";
        const c = render(
          <Errors.Fallback error={errorWithStack} resetErrorBoundary={mockReset} />,
        );
        fireEvent.click(c.getByText("Copy diagnostics"));
        await waitFor(() => {
          expect(writeTextMock).toHaveBeenCalled();
        });
        const copiedText = writeTextMock.mock.calls[0][0];
        expect(copiedText).toContain("Stack Trace:");
        expect(copiedText).toContain("at TestFunction");
      });

      it("should include component stack in copied content", async () => {
        const c = render(
          <Errors.Fallback
            error={mockError}
            componentStack={"    at MyComponent\n    at App"}
            resetErrorBoundary={mockReset}
          />,
        );
        fireEvent.click(c.getByText("Copy diagnostics"));
        await waitFor(() => {
          expect(writeTextMock).toHaveBeenCalled();
        });
        const copiedText = writeTextMock.mock.calls[0][0];
        expect(copiedText).toContain("Component Stack:");
        expect(copiedText).toContain("at MyComponent");
      });

      it("should include extraInfo in copied content", async () => {
        const c = render(
          <Errors.Fallback
            error={mockError}
            resetErrorBoundary={mockReset}
            extraInfo={{ version: "1.0.0", userId: "123" }}
          />,
        );
        fireEvent.click(c.getByText("Copy diagnostics"));
        await waitFor(() => {
          expect(writeTextMock).toHaveBeenCalled();
        });
        const copiedText = writeTextMock.mock.calls[0][0];
        expect(copiedText).toContain("Additional Info:");
        expect(copiedText).toContain('"version": "1.0.0"');
        expect(copiedText).toContain('"userId": "123"');
      });

      it("should not include extraInfo section when extraInfo is empty", async () => {
        const c = render(
          <Errors.Fallback
            error={mockError}
            resetErrorBoundary={mockReset}
            extraInfo={{}}
          />,
        );
        fireEvent.click(c.getByText("Copy diagnostics"));
        await waitFor(() => {
          expect(writeTextMock).toHaveBeenCalled();
        });
        const copiedText = writeTextMock.mock.calls[0][0];
        expect(copiedText).not.toContain("Additional Info:");
      });

      it("should show check icon after copying", async () => {
        const c = render(
          <Errors.Fallback error={mockError} resetErrorBoundary={mockReset} />,
        );
        expect(c.container.querySelector(".pluto-icon--copy")).toBeTruthy();
        expect(c.container.querySelector(".pluto-icon--check")).toBeFalsy();
        fireEvent.click(c.getByText("Copy diagnostics"));
        await waitFor(() => {
          expect(c.container.querySelector(".pluto-icon--check")).toBeTruthy();
        });
        expect(c.container.querySelector(".pluto-icon--copy")).toBeFalsy();
      });
    });
  });
});
