// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { waitForStreamLive } from "@synnaxlabs/client/testutil";
import { render } from "@testing-library/react";
import { type ReactNode } from "react";

import { Notifications } from "@/app/notifications";
import { settleWrites, UncappedFeed } from "@/platform/notifications/testutil";
import { createConsoleWrapper } from "@/testutil";

/**
 * Renders the UI next to {@link UncappedFeed} inside {@link Notifications.Provider}.
 * Waits until the forwarder has loaded the open problems. A problem that starts before
 * the load ends counts as open at load.
 */
export const renderForward = async (client: Client, ui?: ReactNode): Promise<void> => {
  await waitForStreamLive(client.connection);
  const { wrapper } = await createConsoleWrapper({ client });
  render(
    <Notifications.Provider>
      <UncappedFeed />
      {ui}
    </Notifications.Provider>,
    { wrapper },
  );
  await settleWrites(client);
};
