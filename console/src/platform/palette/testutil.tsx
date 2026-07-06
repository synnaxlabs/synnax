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
import { type FC, type PropsWithChildren, type ReactElement } from "react";

import { Export } from "@/platform/export";
import { Import } from "@/platform/import";
import { Modals } from "@/platform/modals";
import { Ontology } from "@/platform/ontology";
import { createServices } from "@/platform/ontology/testutil";
import { Palette } from "@/platform/palette";
import {
  type ConsolePreloadedState,
  createConsoleWrapper,
  type TestStore,
} from "@/testutil";

export interface CreatePaletteWrapperArgs {
  commands?: Palette.Command[];
  client?: Client | null;
  services?: Ontology.Services;
  preloadedState?: ConsolePreloadedState;
}

/**
 * Layers a {@link Palette.CommandProvider} carrying the given commands over the full
 * console provider stack, so palette hooks and the palette UI resolve their commands,
 * store, client, and modals exactly as they do in the app. Returns the wrapper and the
 * backing store.
 */
export const createPaletteWrapper = async ({
  commands = [],
  client = null,
  services = createServices(),
  preloadedState,
}: CreatePaletteWrapperArgs = {}): Promise<{
  wrapper: FC<PropsWithChildren>;
  store: TestStore;
}> => {
  const { wrapper: Base, store } = await createConsoleWrapper({
    client,
    preloadedState,
  });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Base>
      <Ontology.ServicesProvider services={services}>
        <Palette.CommandProvider commands={commands}>
          {children}
        </Palette.CommandProvider>
      </Ontology.ServicesProvider>
    </Base>
  );
  Wrapper.displayName = "PaletteWrapper";
  return { wrapper: Wrapper, store };
};

const TRIGGER_CONFIG: Palette.TriggerConfig = {
  command: [["Control", "Shift", "P"]],
  defaultMode: "command",
  search: [["Control", "P"]],
};

export interface PaletteHandle {
  store: TestStore;
  /** Opens the palette in command mode and types text after the command symbol. */
  openCommandPalette: (text?: string) => Promise<void>;
  /** Waits for the named command to appear in the open palette and selects it. */
  selectCommand: (name: string) => Promise<void>;
}

export interface RenderPaletteArgs extends CreatePaletteWrapperArgs {
  /** Registered file ingesters, for commands that import components. */
  fileIngesters?: Import.FileIngesters;
  /** Registered extractors, for commands that export components. */
  extractors?: Export.Extractors;
}

/**
 * Renders the full palette UI plus a live modal stack over the palette wrapper, so a
 * spec can drive a domain's COMMANDS end to end: open, filter, select, and interact
 * with any modal the command opens. Specs rendering the palette must opt in to
 * stubGeometry() for the virtualized command list.
 */
export const renderPalette = async ({
  fileIngesters = {},
  extractors = {},
  ...args
}: RenderPaletteArgs = {}): Promise<PaletteHandle> => {
  const { wrapper, store } = await createPaletteWrapper(args);
  render(
    <Import.FileIngestersProvider fileIngesters={fileIngesters}>
      <Export.ExtractorsProvider extractors={extractors}>
        <Palette.Palette commandSymbol=">" triggerConfig={TRIGGER_CONFIG} />
        <Modals.Stack />
      </Export.ExtractorsProvider>
    </Import.FileIngestersProvider>,
    { wrapper },
  );
  const openCommandPalette = async (text = ""): Promise<void> => {
    const btn = document.querySelector<HTMLElement>(".console-palette__btn");
    if (btn == null) throw new Error("palette open button not found");
    fireEvent.click(btn);
    const input = await waitFor(() => {
      const el = document.querySelector<HTMLInputElement>("input");
      if (el == null) throw new Error("palette input not found");
      return el;
    });
    fireEvent.change(input, { target: { value: `>${text}` } });
  };
  const selectCommand = async (name: string): Promise<void> => {
    const item = await screen.findByText(name);
    // detail: 1 marks the click as a real pointer click; the list otherwise fires the
    // selection twice (keyboard + pointer paths).
    await act(async () => {
      fireEvent.click(item, { detail: 1 });
    });
  };
  return { store, openCommandPalette, selectCommand };
};
