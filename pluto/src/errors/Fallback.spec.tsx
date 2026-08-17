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
import { z } from "zod";

import { Errors } from "@/errors";
import { resolveStack } from "@/errors/resolveStack";

// Resolution fetches scripts and source maps over the network; stub the resolveStack
// module so Fallback's resolution effect is deterministic. Stacks containing the
// RESOLVABLE_STACK / RESOLVABLE_COMPONENT tokens resolve; anything else degrades to
// its raw input, mirroring the real module's never-rejecting contract. Rejection
// itself is exercised via mockRejectedValueOnce in the dedicated failure test.
vi.mock("@/errors/resolveStack", () => ({
  resolveStack: vi.fn(async (error: Error, componentStack: string | null) => ({
    stack:
      error.stack === "RESOLVABLE_STACK"
        ? "  at resolvedFn (src/resolved.ts:99:4)"
        : (error.stack ?? ""),
    componentStack:
      componentStack === "RESOLVABLE_COMPONENT"
        ? "  at ResolvedComponent (src/Resolved.tsx:12)"
        : componentStack,
  })),
}));

describe("Fallback", () => {
  let warnSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    // Fallback warns on source-map resolution failure; silence by default and assert
    // against it in the dedicated failure test.
    warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
  });

  afterEach(() => {
    warnSpy.mockRestore();
  });

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

  describe("message formatting", () => {
    const schema = z.object({
      nodes: z.array(z.object({ color: z.tuple([z.number(), z.number().max(1)]) })),
    });
    const zodError = schema.safeParse({ nodes: [{ color: [0, 255] }] }).error!;

    it("should render a prettified zod error instead of the raw issues JSON", () => {
      const c = render(
        <Errors.Fallback
          error={zodError as unknown as Error}
          resetErrorBoundary={mockReset}
        />,
      );
      const message = c.container.querySelector(".pluto-error-fallback__message");
      expect(message?.textContent).toContain("at nodes[0].color[1]");
      expect(message?.textContent).not.toContain('"code"');
    });

    it("should mark a multiline message so it renders as a scrollable block", () => {
      const c = render(
        <Errors.Fallback
          error={zodError as unknown as Error}
          resetErrorBoundary={mockReset}
        />,
      );
      expect(
        c.container.querySelector(".pluto-error-fallback__message--multiline"),
      ).toBeTruthy();
    });

    it("should not mark a single-line message as multiline", () => {
      const c = render(
        <Errors.Fallback error={mockError} resetErrorBoundary={mockReset} />,
      );
      expect(
        c.container.querySelector(".pluto-error-fallback__message--multiline"),
      ).toBeFalsy();
    });

    it("should keep the raw zod message in copy diagnostics", async () => {
      const writeTextMock = vi.fn().mockResolvedValue(undefined);
      Object.assign(navigator, { clipboard: { writeText: writeTextMock } });
      const c = render(
        <Errors.Fallback
          error={zodError as unknown as Error}
          resetErrorBoundary={mockReset}
        />,
      );
      fireEvent.click(c.getByText("Copy diagnostics"));
      await waitFor(() => {
        expect(writeTextMock).toHaveBeenCalled();
      });
      expect(writeTextMock.mock.calls[0][0]).toContain('"code"');
    });
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

  describe("Copy diagnostics", () => {
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
      expect(copiedText).toContain("Stack trace:");
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
      expect(copiedText).toContain("Component stack:");
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
      expect(copiedText).toContain("Additional info:");
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
      expect(copiedText).not.toContain("Additional info:");
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

  describe("source-map resolution", () => {
    let writeTextMock: ReturnType<typeof vi.fn>;

    beforeEach(() => {
      writeTextMock = vi.fn().mockResolvedValue(undefined);
      Object.assign(navigator, {
        clipboard: { writeText: writeTextMock },
      });
    });

    it("should swap raw stack for the resolved version once resolution completes", async () => {
      const errorWithStack = new Error("Boom");
      errorWithStack.stack = "RESOLVABLE_STACK";
      const c = render(
        <Errors.Fallback
          error={errorWithStack}
          componentStack={null}
          resetErrorBoundary={mockReset}
        />,
      );
      // The raw (mock) stack token renders first, then is replaced by the resolved
      // frame after the effect runs.
      await waitFor(() => {
        expect(c.container.textContent).toContain(
          "at resolvedFn (src/resolved.ts:99:4)",
        );
      });
      expect(c.container.textContent).not.toContain("RESOLVABLE_STACK");
      expect(warnSpy).not.toHaveBeenCalled();
    });

    it("should prefer resolved component stack over resolved error stack", async () => {
      const errorWithStack = new Error("Boom");
      errorWithStack.stack = "RESOLVABLE_STACK";
      const c = render(
        <Errors.Fallback
          error={errorWithStack}
          componentStack="RESOLVABLE_COMPONENT"
          resetErrorBoundary={mockReset}
        />,
      );
      await waitFor(() => {
        expect(c.container.textContent).toContain(
          "at ResolvedComponent (src/Resolved.tsx:12)",
        );
      });
      expect(warnSpy).not.toHaveBeenCalled();
    });

    it("should keep the raw stack and warn when resolution rejects", async () => {
      vi.mocked(resolveStack).mockRejectedValueOnce(new Error("no maps in test env"));
      const errorWithStack = new Error("Boom");
      errorWithStack.stack = "Error: Boom\n    at unresolvable (x.js:1:1)";
      const c = render(
        <Errors.Fallback
          error={errorWithStack}
          componentStack={null}
          resetErrorBoundary={mockReset}
        />,
      );
      // resolveStack never rejects in production; an unexpected rejection keeps the
      // raw stack and is surfaced via console.warn rather than crashing the crash
      // screen.
      await waitFor(() => {
        expect(warnSpy).toHaveBeenCalledTimes(1);
      });
      expect(c.container.textContent).toContain("at unresolvable (x.js:1:1)");
      const [message, cause] = warnSpy.mock.calls[0];
      expect(message).toBe("Unexpected source-map resolution failure");
      expect(cause).toBeInstanceOf(Error);
      expect((cause as Error).message).toBe("no maps in test env");
    });

    it("should include the resolved stack in copy diagnostics output", async () => {
      const errorWithStack = new Error("Boom");
      errorWithStack.name = "BoomError";
      errorWithStack.stack = "RESOLVABLE_STACK";
      const c = render(
        <Errors.Fallback error={errorWithStack} resetErrorBoundary={mockReset} />,
      );
      await waitFor(() => {
        expect(c.container.textContent).toContain("at resolvedFn");
      });
      fireEvent.click(c.getByText("Copy diagnostics"));
      await waitFor(() => {
        expect(writeTextMock).toHaveBeenCalled();
      });
      const copied = writeTextMock.mock.calls[0][0];
      expect(copied).toContain("Stack trace:\n  at resolvedFn (src/resolved.ts:99:4)");
      expect(copied).not.toContain("RESOLVABLE_STACK");
      expect(warnSpy).not.toHaveBeenCalled();
    });
  });
});
