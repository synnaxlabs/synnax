// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status, type Synnax as Client } from "@synnaxlabs/client";
import { waitForStreamLive } from "@synnaxlabs/client/testutil";
import { TimeStamp } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { type ReactElement, type ReactNode } from "react";

import { Notifications } from "@/app/notifications";
import { Notifications as PlatformNotifications } from "@/platform/notifications";
import { settleWrites, UncappedFeed } from "@/platform/notifications/testutil";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const Forward = (): ReactElement => {
  Notifications.useForward();
  return <UncappedFeed />;
};

/**
 * Runs {@link Notifications.useForward} inside a notifications provider, and renders
 * the UI next to {@link UncappedFeed}. Waits until the forwarder has loaded the open
 * problems. A problem that starts before the load ends counts as open at load.
 */
export const renderForward = async (client: Client, ui?: ReactNode): Promise<void> => {
  await waitForStreamLive(client.connection);
  const { wrapper } = await createConsoleWrapper({ client });
  render(
    <PlatformNotifications.Provider>
      <Forward />
      {ui}
    </PlatformNotifications.Provider>,
    { wrapper },
  );
  await settleWrites(client);
};

/** @returns the message of a new status on the key. */
export const writeStatus = async (
  client: Client,
  key: status.Key,
  variant: status.Variant,
  details?: unknown,
): Promise<string> => {
  const { message } = await client.statuses.set({
    key,
    name: "",
    variant,
    message: uniqueName("message"),
    time: TimeStamp.now(),
    details,
  });
  return message;
};
