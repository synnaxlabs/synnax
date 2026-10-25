// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type device, type rack } from "@synnaxlabs/client";
import { z } from "zod";

import { type Tree } from "@/platform/tree";
import { type Session } from "@/session";

/** Configure-modal params; the key is required since it edits an existing device. */
export interface ConfigureParams {
  deviceKey: device.Key;
}

/**
 * Like {@link ConfigureParams}, but the key is optional since connect can also create a
 * new connection.
 */
export interface ConnectParams extends Partial<ConfigureParams> {
  /** The rack a new device starts with in its "Connect from" field. */
  rack?: rack.Key;
}

/** Props of a connect form. The form closes with the key of the device it saved. */
export interface ConnectProps extends Session.Modals.ContentProps<
  ConnectParams,
  device.Key
> {
  /** Returns to the picker that opened the form. Absent when the form opens alone. */
  onBack?: () => void;
}

/** Props of an integration's device context menu items. */
export interface ContextMenuItemsProps extends Tree.ContextMenuProps {
  /** Opens the configure or connect form of a device. */
  onConfigure: (deviceKey: device.Key) => void;
}

const IDENTIFIER_MESSAGE = "Identifier must be between 2-12 characters";

export const nameZ = z.string().min(1, "Name is required");

export const identifierZ = z
  .string()
  .min(2, IDENTIFIER_MESSAGE)
  .max(12, IDENTIFIER_MESSAGE)
  .regex(
    /^[a-zA-Z][a-zA-Z0-9_]*$/,
    "Identifier must start with a letter and contain only letters, numbers, and underscores",
  );

export const commandStatePairZ = z.object({
  command: channel.keyZ,
  state: channel.keyZ,
});

export type CommandStatePair = z.infer<typeof commandStatePairZ>;
export const ZERO_COMMAND_STATE_PAIR: CommandStatePair = { command: 0, state: 0 };
