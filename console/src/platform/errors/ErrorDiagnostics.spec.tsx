// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, panel, status } from "@synnaxlabs/client";
import { createPanelParent, createTestClient } from "@synnaxlabs/client/testutil";
import { Unreachable } from "@synnaxlabs/freighter";
import { Errors } from "@synnaxlabs/pluto";
import { render, screen } from "@testing-library/react";
import { act, type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Boundary } from "@/platform/errors/Boundary";
import { ErrorDiagnostics } from "@/platform/errors/ErrorDiagnostics";
import { Session } from "@/session";
import {
  createConsoleWrapper,
  createTestStore,
  renderWithConsole,
  type TestStore,
} from "@/testutil";

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

  it("appends the crashed panel's name and key", async () => {
    const client = createTestClient();
    const { wrapper, store } = await createConsoleWrapper({ client });
    void act(() => store.dispatch(Session.Cluster.select("LOCAL")));
    const doc = panel.panelZ.parse({
      name: "fridge_schem",
      root: { variant: "leaf", tabs: [] },
    });
    await client.panels.create({ ...doc, parent: await createPanelParent(client) });
    // Prime the query cache the way the mosaic's retrieve does.
    await client.panels.retrieve(doc.key);
    render(
      <Boundary panelKey={doc.key}>
        <Throw error={retrieveNotFoundError()} />
      </Boundary>,
      { wrapper },
    );
    const expected = [
      "Failed to retrieve schematic",
      "Core: Local (localhost:9090)",
      `"fridge_schem" (${doc.key})`,
    ].join("\n");
    expect(messageText()).toBe(expected);
  });

  it("replaces the crash page when a flux read failed on an unreachable Core", async () => {
    const store = await createTestStore();
    void act(() => store.dispatch(Session.Cluster.select("LOCAL")));
    const error = new Error("Failed to retrieve channel group", {
      cause: new Unreachable(),
    });
    await renderBoundary(store, <Throw error={error} />);
    expect(screen.getByText("Local")).toBeTruthy();
    expect(screen.getByText("localhost:9090")).toBeTruthy();
    expect(messageText()).toBeNull();
  });

  it("finds an unreachable Core nested under a status-shaped cause", async () => {
    const store = await createTestStore();
    void act(() => store.dispatch(Session.Cluster.select("LOCAL")));
    const error = status.toError(
      status.fromException(new Unreachable(), "Failed to retrieve channel group"),
    );
    await renderBoundary(store, <Throw error={error} />);
    expect(screen.getByText("Local")).toBeTruthy();
    expect(messageText()).toBeNull();
  });

  it("keeps the crash page for a failure the Core answered", async () => {
    const store = await createTestStore();
    void act(() => store.dispatch(Session.Cluster.select("LOCAL")));
    await renderBoundary(store, <Throw error={retrieveNotFoundError()} />);
    expect(screen.queryByText("localhost:9090")).toBeNull();
    expect(messageText()).toBe(
      "Failed to retrieve schematic\nCore: Local (localhost:9090)",
    );
  });

  it("omits the name when the crashed panel isn't cached", async () => {
    const store = await createTestStore();
    await renderWithConsole(
      <Boundary panelKey="uncached-key">
        <Throw error={new Error("boom")} />
      </Boundary>,
      { store },
    );
    expect(messageText()).toBe("boom\nCore: none\n(uncached-key)");
  });
});
