// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { render } from "@testing-library/react";
import { type ReactNode } from "react";

import { Modals } from "@/platform/modals";
import { createConsoleWrapper } from "@/testutil";

export const client = createTestClient();

/**
 * Renders ui together with a live {@link Modals.Stack} inside the full console provider
 * stack backed by a real cluster client, so modal-driven arc flows (create, rename)
 * actually mount and hit the server. Returns the render result plus the backing store.
 */
export const renderArc = async (ui: ReactNode) => {
  const { wrapper, store } = await createConsoleWrapper({ client });
  return {
    ...render(
      <>
        {ui}
        <Modals.Stack />
      </>,
      { wrapper },
    ),
    store,
  };
};
