// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, status } from "@synnaxlabs/client";
import { Errors } from "@synnaxlabs/pluto";
import { act, type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ErrorDiagnostics } from "@/platform/errors/ErrorDiagnostics";
import { Layout } from "@/platform/layout";
import { placeLayout } from "@/platform/layout/testutil";
import { Session } from "@/session";
import { createTestStore, renderWithConsole, type TestStore } from "@/testutil";

// Errors.Fallback kicks off a source-map resolution effect; stub stacktrace-js at the
// library boundary so it doesn't hit the network.
vi.mock("stacktrace-js", () => ({
  default: {
    fromError: async () => {
      throw new Error("no maps in test env");
    },
  },
}));

const Throw = ({ error }: { error: Error }): ReactElement => {
  throw error;
};

const retrieveNotFoundError = (): Error =>
  status.toError(
    status.fromException(
      new NotFoundError("schematic l1 not found"),
      "Failed to retrieve schematic",
    ),
  );

const messageText = (): string | null =>
  document.querySelector(".pluto-error-fallback__message")?.textContent ?? null;

describe("ErrorDiagnostics", () => {
  let consoleSpy: ReturnType<typeof vi.spyOn>;
  let warnSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
  });

  afterEach(() => {
    consoleSpy.mockRestore();
    warnSpy.mockRestore();
  });

  const renderBoundary = async (store: TestStore, child: ReactElement) =>
    await renderWithConsole(
      <Errors.Boundary FallbackComponent={ErrorDiagnostics}>{child}</Errors.Boundary>,
      { store },
    );

  it("appends the connected Core to any error", async () => {
    const store = await createTestStore();
    void act(() => store.dispatch(Session.Cluster.select("LOCAL")));
    await renderBoundary(store, <Throw error={new Error("boom")} />);
    expect(messageText()).toBe("boom\nCore: Local (localhost:9090)");
  });

  it("reports Core none when not connected to a Core", async () => {
    const store = await createTestStore();
    await renderBoundary(store, <Throw error={new Error("boom")} />);
    expect(messageText()).toBe("boom\nCore: none");
  });

  it("preserves the original error name in the fallback", async () => {
    const store = await createTestStore();
    const error = new Error("boom");
    error.name = "NotFoundError";
    await renderBoundary(store, <Throw error={error} />);
    const nameText = document.querySelector(".pluto-error-fallback__name")?.textContent;
    expect(nameText).toBe("NotFoundError");
    expect(messageText()).toBe("boom\nCore: none");
  });

  it("appends page name and key for a layout page crash", async () => {
    const store = await createTestStore();
    placeLayout(store, "l1", { type: "schematic", name: "fridge_schem" });
    void act(() => store.dispatch(Session.Cluster.select("LOCAL")));
    const Renderer: Layout.Renderer = () => {
      throw retrieveNotFoundError();
    };
    Renderer.displayName = "ThrowingRenderer";
    await renderWithConsole(
      <Layout.RendererProvider value={{ schematic: Renderer }}>
        <Layout.Content layoutKey="l1" />
      </Layout.RendererProvider>,
      { store },
    );
    const expected = [
      "Failed to retrieve schematic",
      "Core: Local (localhost:9090)",
      '"fridge_schem" (l1)',
    ].join("\n");
    expect(messageText()).toBe(expected);
  });

  it("omits the name when a layout page has no name", async () => {
    const store = await createTestStore();
    const renderFallback = (props: Errors.FallbackProps): ReactElement => (
      <ErrorDiagnostics page={{ key: "l1" }} {...props} />
    );
    await renderWithConsole(
      <Errors.Boundary FallbackComponent={renderFallback}>
        <Throw error={new Error("boom")} />
      </Errors.Boundary>,
      { store },
    );
    expect(messageText()).toBe("boom\nCore: none\n(l1)");
  });
});
