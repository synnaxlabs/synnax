// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/x";
import { act, render } from "@testing-library/react";
import { type ReactElement, use } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { type FallbackProps } from "@/errors/Fallback";
import { SuspenseBoundary } from "@/errors/SuspenseBoundary";

const Suspender = ({ promise }: { promise: Promise<string> }): ReactElement => (
  <div>{use(promise)}</div>
);

// Test helper that deliberately throws arbitrary values to exercise SuspenseBoundary's
// fallback path with non-Error rejections.
const Thrower = ({ throwable }: { throwable: unknown }): ReactElement => {
  // eslint-disable-next-line @typescript-eslint/only-throw-error
  throw throwable;
};

describe("SuspenseBoundary", () => {
  let consoleSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    consoleSpy.mockRestore();
  });

  it("should render children when nothing suspends or throws", () => {
    const c = render(
      <SuspenseBoundary>
        <div>resolved</div>
      </SuspenseBoundary>,
    );
    expect(c.getByText("resolved")).toBeTruthy();
  });

  it("should render the loading fallback while a child suspends", () => {
    let resolve!: (value: string) => void;
    const promise = new Promise<string>((r) => {
      resolve = r;
    });
    const c = render(
      <SuspenseBoundary loading={<div>loading</div>}>
        <Suspender promise={promise} />
      </SuspenseBoundary>,
    );
    expect(c.getByText("loading")).toBeTruthy();
    resolve("done");
  });

  it("should render the default Fallback when a child throws an Error", () => {
    const c = render(
      <SuspenseBoundary>
        <Thrower throwable={new Error("boom")} />
      </SuspenseBoundary>,
    );
    expect(c.getByText("boom")).toBeTruthy();
    expect(c.getByText("Reload")).toBeTruthy();
  });

  it("should render a custom FallbackComponent when a child throws", () => {
    const Fallback = ({ error }: FallbackProps) => (
      <div data-testid="custom">{error.message}</div>
    );
    const c = render(
      <SuspenseBoundary FallbackComponent={Fallback}>
        <Thrower throwable={new Error("boom")} />
      </SuspenseBoundary>,
    );
    expect(c.getByTestId("custom").textContent).toBe("boom");
  });

  it("should expose the original status on error.cause when a thrown Error wraps one", () => {
    const inner = new Error("raw");
    const s = status.fromException(inner, "Failed to fetch");
    let received!: FallbackProps["error"];
    const Fallback = ({ error }: FallbackProps) => {
      received = error;
      return <div>caught</div>;
    };
    render(
      <SuspenseBoundary FallbackComponent={Fallback}>
        <Thrower throwable={status.toError(s)} />
      </SuspenseBoundary>,
    );
    expect(received.message).toBe("Failed to fetch");
    expect(received.cause).toBe(s);
  });

  it("should reset and re-render children when resetErrorBoundary is called", async () => {
    let shouldThrow = true;
    const Maybe = (): ReactElement => {
      if (shouldThrow) throw new Error("boom");
      return <div>recovered</div>;
    };
    const Fallback = ({ resetErrorBoundary }: FallbackProps) => (
      <button onClick={resetErrorBoundary}>retry</button>
    );
    const c = render(
      <SuspenseBoundary FallbackComponent={Fallback}>
        <Maybe />
      </SuspenseBoundary>,
    );
    expect(c.getByText("retry")).toBeTruthy();
    shouldThrow = false;
    await act(async () => {
      c.getByText("retry").click();
    });
    expect(c.getByText("recovered")).toBeTruthy();
  });
});
