// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { screen } from "@testing-library/react";
import { act, type ReactElement, use } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Toolbar } from "@/platform/toolbar";
import { renderWithConsole } from "@/testutil";

// Errors.Fallback kicks off a source-map resolution effect; stub stacktrace-js at the
// library boundary so it doesn't hit the network.
vi.mock("stacktrace-js", () => ({
  default: {
    fromError: async () => {
      throw new Error("no maps in test env");
    },
  },
}));

const Read = ({ promise }: { promise: Promise<void> }): ReactElement => {
  use(promise);
  return <p>body</p>;
};

const Throw = (): ReactElement => {
  throw new Error("boom");
};

const renderToolbar = async (body: ReactElement) =>
  await renderWithConsole(
    <Toolbar.Content>
      <Toolbar.Header>
        <Toolbar.Title>Ranges</Toolbar.Title>
      </Toolbar.Header>
      <Toolbar.Body>{body}</Toolbar.Body>
    </Toolbar.Content>,
  );

describe("Toolbar.Body", () => {
  let consoleSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => consoleSpy.mockRestore());

  it("keeps the header painted while the body's read is pending", async () => {
    let resolve!: () => void;
    const promise = new Promise<void>((r) => (resolve = r));
    await renderToolbar(<Read promise={promise} />);
    expect(screen.getByText("Ranges")).toBeTruthy();
    expect(screen.queryByText("body")).toBeNull();
    await act(async () => resolve());
    expect(screen.getByText("body")).toBeTruthy();
  });

  it("keeps the header painted when the body crashes", async () => {
    await renderToolbar(<Throw />);
    expect(screen.getByText("Ranges")).toBeTruthy();
    expect(screen.getByText("boom", { exact: false })).toBeTruthy();
  });
});
