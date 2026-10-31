// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack, type status, type Synnax } from "@synnaxlabs/client";
import { waitForStreamLive } from "@synnaxlabs/client/testutil";
import { Status } from "@synnaxlabs/lyra/status";
import { TimeStamp } from "@synnaxlabs/x";
import {
  fireEvent,
  renderHook,
  type RenderHookResult,
  screen,
  within,
} from "@testing-library/react";
import { type PropsWithChildren, type ReactElement, type ReactNode } from "react";

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
        <Status.Notification key={status.id} status={status} silence={silence} />
      ))}
    </>
  );
};

/**
 * Writes a status with a unique message.
 * @returns the message of the status.
 */
export const writeStatus = async (
  client: Synnax,
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

/** @returns whether the rule holds the thing with the key in use. */
export const checkInUse = async (
  client: Synnax,
  { inUse }: Notifications.Rule,
  key: string,
): Promise<boolean> => {
  if (inUse == null) throw new Error("the rule must check use");
  return await inUse({ client, key });
};

/**
 * Writes an alarm and waits until it notifies. The Core streams statuses in write
 * order, so the forwarder has received every status written before it.
 */
export const settleWrites = async (client: Synnax): Promise<void> => {
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

/**
 * @returns the key of a rack that the Core no longer has. The Core writes problems of
 * its own on the key of each rack that exists, but none on this key.
 */
export const createRackKey = async (client: Synnax): Promise<rack.Key> => {
  const { key } = await client.racks.create({ name: uniqueName("rack") });
  await client.racks.delete(key);
  return key;
};

/** @returns the notification that shows the message. */
export const findNotification = async (message: string): Promise<HTMLElement> => {
  const notification = (await screen.findByText(message)).closest<HTMLElement>(
    "[role='status']",
  );
  if (notification == null) throw new Error(`no notification for ${message}`);
  return notification;
};

/** Closes the notification that shows the message. */
export const close = async (message: string): Promise<void> => {
  const notification = await findNotification(message);
  fireEvent.click(within(notification).getByRole("button", { name: "Silence" }));
};

/**
 * Renders the UI above {@link UncappedFeed} inside a {@link Notifications.Provider}
 * with the rules. Waits until the forwarder has loaded the open problems. A problem
 * that starts before the load ends counts as open at load.
 * @returns the result of {@link Notifications.useProblems}.
 */
export const renderForward = async (
  client: Synnax,
  rules: Notifications.Rules,
  ui?: ReactNode,
): Promise<RenderHookResult<Notifications.Problem[], unknown>> => {
  await waitForStreamLive(client.connection);
  const { wrapper: Console } = await createConsoleWrapper({ client });
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Notifications.Provider rules={rules}>
        {children}
        {ui}
        <UncappedFeed />
      </Notifications.Provider>
    </Console>
  );
  const rendered = renderHook(Notifications.useProblems, { wrapper });
  await settleWrites(client);
  return rendered;
};
