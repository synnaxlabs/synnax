// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { type FC, type PropsWithChildren, type ReactElement } from "react";

import { Ontology } from "@/platform/ontology";
import { Palette } from "@/platform/palette";
import { createConsoleWrapper, type TestStore } from "@/testutil";

const EMPTY_SERVICES = {} as Ontology.Services;

export const TIMEOUT = { timeout: 5000 };

export interface CreatePaletteWrapperArgs {
  commands?: Palette.Command[];
  client?: Client | null;
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
}: CreatePaletteWrapperArgs = {}): Promise<{
  wrapper: FC<PropsWithChildren>;
  store: TestStore;
}> => {
  const { wrapper: Base, store } = await createConsoleWrapper({ client });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Base>
      <Ontology.ServicesProvider services={EMPTY_SERVICES}>
        <Palette.CommandProvider commands={commands}>
          {children}
        </Palette.CommandProvider>
      </Ontology.ServicesProvider>
    </Base>
  );
  Wrapper.displayName = "PaletteWrapper";
  return { wrapper: Wrapper, store };
};
