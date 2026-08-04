// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement, useState } from "react";

import { Command } from "@/feature/command";
import { Import } from "@/platform/import";
import { Modals } from "@/platform/modals";
import {
  type ConsolePreloadedState,
  createConsoleWrapper,
  type TestStore,
} from "@/testutil";

const COMMAND_PREFIX = ">";
const PLACEHOLDER = <>Type {COMMAND_PREFIX} to view commands</>;

export interface RenderPaletteParams {
  commands?: Command.Command[];
  client?: Client | null;
  preloadedState?: ConsolePreloadedState;
  /** Registered file ingesters, for commands that import components. */
  fileIngesters?: Import.FileIngesters;
}

export interface PaletteHandle {
  store: TestStore;
  /** Sets the palette input to command mode and filters by the given text. */
  openCommandPalette: (text?: string) => Promise<void>;
  /** Waits for the named command to appear and selects it. */
  selectCommand: (name: string) => Promise<void>;
}

const paletteInput = (): HTMLInputElement => {
  const input = document.querySelector<HTMLInputElement>(
    ".console-palette__input input",
  );
  if (input == null) throw new Error("palette input not found");
  return input;
};

interface CommandPaletteProps {
  commands: Command.Command[];
}

const CommandPalette = ({ commands }: CommandPaletteProps): ReactElement => {
  const [value, setValue] = useState(COMMAND_PREFIX);
  return (
    <Command.List
      commands={commands}
      value={value}
      inputPlaceholder={PLACEHOLDER}
      onChange={setValue}
    />
  );
};

/**
 * Renders a domain's COMMANDS through the real command list plus a live modal stack, so
 * a spec can drive them end to end: filter, select, and interact with any modal a
 * command opens. Only commands are wired here; resource search lives in a separate seam.
 */
export const renderPalette = async ({
  commands = [],
  client = null,
  preloadedState,
  fileIngesters = {},
}: RenderPaletteParams = {}): Promise<PaletteHandle> => {
  const { wrapper, store } = await createConsoleWrapper({ client, preloadedState });
  render(
    <Import.FileIngestersProvider fileIngesters={fileIngesters}>
      <CommandPalette commands={commands} />
      <Modals.Stack />
    </Import.FileIngestersProvider>,
    { wrapper },
  );
  const openCommandPalette = async (text = ""): Promise<void> => {
    await waitFor(paletteInput);
    fireEvent.change(paletteInput(), {
      target: { value: `${COMMAND_PREFIX}${text}` },
    });
  };
  const selectCommand = async (name: string): Promise<void> => {
    const item = await screen.findByText(name);
    // detail: 1 marks a real pointer click; the select frame converts it into the
    // synthetic detail-0 click that invokes the command, so it fires exactly once.
    await act(async () => {
      fireEvent.click(item, { detail: 1 });
    });
  };
  return { store, openCommandPalette, selectCommand };
};
