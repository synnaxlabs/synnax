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
import { type PropsWithChildren, type ReactElement } from "react";

import { Guard } from "@/feature/license/Guard";
import { createConsoleWrapper, type TestStore } from "@/testutil";

/** The text the guarded children render, so a spec can tell whether they show. */
export const GUARDED_CONTENT = "licensed content";

/**
 * Renders a {@link Guard} over {@link GUARDED_CONTENT} with the given client and
 * connection status, and returns the backing store. The client is handed to the
 * provider unconnected: a Core that refuses requests for want of a license never
 * settles a connection, so the screen must not wait on one.
 */
export const renderGuard = async (
  client: Client | null,
  status?: connection.Status,
): Promise<TestStore> => {
  const { wrapper: Console, store } = await createConsoleWrapper({ client: null });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Synnax.TestProvider client={client} status={status}>
        {children}
      </Synnax.TestProvider>
    </Console>
  );
  Wrapper.displayName = "GuardWrapper";
  render(
    <Guard>
      <span>{GUARDED_CONTENT}</span>
    </Guard>,
    { wrapper: Wrapper },
  );
  return store;
};
