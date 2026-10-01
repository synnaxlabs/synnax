// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type connection, type Synnax as Client } from "@synnaxlabs/client";
import { Synnax } from "@synnaxlabs/pluto";
import { render } from "@testing-library/react";
import { type ReactElement } from "react";

import { Guard } from "@/feature/license/Guard";
import { createConsoleWrapper, type TestStore } from "@/testutil";

export const GUARDED_CONTENT = "licensed content";

export interface GuardHarness {
  store: TestStore;
  /** Changes the connection status the Guard sees. */
  setStatus: (status?: connection.Status) => void;
}

/**
 * Renders a {@link Guard} over {@link GUARDED_CONTENT} with the given client and
 * connection status. The client stays unconnected, because an unlicensed Core never
 * settles a connection.
 */
export const renderGuard = async (
  client: Client | null,
  status?: connection.Status,
): Promise<GuardHarness> => {
  const { wrapper, store } = await createConsoleWrapper({ client: null });
  const ui = (current?: connection.Status): ReactElement => (
    <Synnax.TestProvider client={client} status={current}>
      <Guard>
        <span>{GUARDED_CONTENT}</span>
      </Guard>
    </Synnax.TestProvider>
  );
  const { rerender } = render(ui(status), { wrapper });
  return { store, setStatus: (next) => rerender(ui(next)) };
};
