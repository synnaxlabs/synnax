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
import { Status } from "@synnaxlabs/lyra/status";
import { TimeStamp } from "@synnaxlabs/x";
import { render, screen } from "@testing-library/react";
import { type ReactElement } from "react";

import { Notifications } from "@/platform/notifications";
import { createConsoleWrapper, uniqueName } from "@/testutil";

/**
 * Renders every notification. {@link Notifications.Feed} shows only four, and specs
 * that share a Core get the status writes of every other spec.
 */
export const UncappedFeed = (): ReactElement => {
  const { statuses, silence } = Status.useNotifications();
  return (
    <>
      {statuses.map((status) => (
        <Status.Notification key={status.key} status={status} silence={silence} />
      ))}
    </>
  );
};

interface ForwardProps {
  rules: Notifications.Rules;
}

const Forward = ({ rules }: ForwardProps): ReactElement => {
  Notifications.useForward(rules);
  return <UncappedFeed />;
};

/**
 * Renders {@link UncappedFeed} with {@link Notifications.useForward} for the rules,
 * once the client streams status writes.
 */
export const renderForward = async (
  client: Client,
  rules: Notifications.Rules,
): Promise<void> => {
  await waitForStreamLive(client.connection);
  const { wrapper } = await createConsoleWrapper({ client });
  render(<Forward rules={rules} />, { wrapper });
};

/**
 * Writes an alarm and waits until it notifies. The Core streams statuses in write
 * order, so the forwarder has received every status written before it.
 */
export const settleWrites = async (client: Client): Promise<void> => {
  const message = uniqueName("settled");
  await client.statuses.set({
    key: uniqueName("alarm"),
    name: "",
    variant: "info",
    message,
    time: TimeStamp.now(),
  });
  await screen.findByText(message);
};
