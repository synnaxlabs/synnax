// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Menu } from "@synnaxlabs/lyra/menu";
import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import type { FC, PropsWithChildren } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { type Locator, page, userEvent } from "vitest/browser";

import { Base } from "@/log/Base";
import { createAetherSetup } from "@/log/testutil";
import { createSynnaxWrapper } from "@/testutil/Synnax";

// Partial Aether mock: only intercepts type "log" for controlling worker-computed
// state (empty, scrolling, selectionStart, etc.). All other Aether consumers
// (Status, Flux, etc.) get the real implementation via the test providers.
// Type assertions below follow existing vi.mock patterns (vitest doesn't expose
// module types from importOriginal without import() annotations, which lint forbids).
const mockAetherUse = vi.hoisted(() => vi.fn());
vi.mock("@/aether", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  const realAether = actual.Aether as Record<string, unknown>;
  return {
    ...actual,
    Aether: {
      ...realAether,
      use: (...args: unknown[]) => {
        const opts = args[0] as { type?: string } | undefined;
        if (opts?.type === "log") return mockAetherUse(...args);
        return (realAether.use as (...a: unknown[]) => unknown)(...args);
      },
    },
  };
});

const SynnaxWrapper = createSynnaxWrapper({ client: null });
const Wrapper: FC<PropsWithChildren> = ({ children }) => (
  <SynnaxWrapper>
    <Triggers.Provider>{children}</Triggers.Provider>
  </SynnaxWrapper>
);

const setupAether = createAetherSetup(mockAetherUse);

const renderLog = async (props: Record<string, unknown> = {}) =>
  await renderThemed(
    <Wrapper>
      <div style={{ width: 400, height: 500 }}>
        <Base {...props} />
      </div>
    </Wrapper>,
  );

const item = (name: string): Locator => page.getByRole("menuitem", { name });

const openMenu = async (): Promise<void> => {
  await userEvent.click(page.getByRole("log"), {
    button: "right",
    position: { x: 10, y: 10 },
  });
};

describe("log/Base context menu", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupAether();
  });

  it("should append the consumer's extra context menu items", async () => {
    await renderLog({
      extraContextMenuItems: <Menu.Item itemKey="extra">Extra</Menu.Item>,
    });
    await openMenu();
    await expect.element(item("Extra")).toBeInTheDocument();
  });

  it("should run undo from the menu when the host supplies the handlers", async () => {
    setupAether({ empty: false });
    const undo = vi.fn();
    const redo = vi.fn();
    await renderLog({ undoRedo: { undo, redo, canUndo: true, canRedo: true } });
    await openMenu();
    // Both entries are asserted before the click, since selecting one closes the menu
    // and takes the other out of the document with it.
    await expect.element(item("Undo")).toBeInTheDocument();
    await expect.element(item("Redo")).toBeInTheDocument();
    await item("Undo").click();
    expect(undo).toHaveBeenCalledTimes(1);
    expect(redo).not.toHaveBeenCalled();
  });

  it("should omit undo and redo when the host supplies none", async () => {
    setupAether({ empty: false });
    await renderLog();
    await openMenu();
    // Positive control: the menu opened, so the absence below is the missing prop
    // rather than a menu that never rendered.
    await expect.element(item("Copy")).toBeInTheDocument();
    await expect.element(item("Undo")).not.toBeInTheDocument();
  });
});
