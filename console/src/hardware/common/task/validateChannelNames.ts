// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, NotFoundError, type Synnax } from "@synnaxlabs/client";
import { primitive } from "@synnaxlabs/x";

/**
 * Determines if a channel needs to be created by checking if it exists in Synnax.
 * Handles both missing keys (not in device properties) and deleted channels.
 *
 * @param client - Synnax client instance
 * @param existingKey - Channel key from device properties (may be 0, null, or undefined)
 * @returns true if channel needs creation, false if it already exists
 *
 */
export const shouldCreateChannel = async (
  client: Synnax,
  existingKey: channel.Key | undefined | null,
): Promise<boolean> => {
  if (primitive.isZero(existingKey)) return true;
  try {
    await client.channels.retrieve(existingKey);
    return false; // Channel exists
  } catch (e) {
    if (NotFoundError.matches(e)) return true; // Channel was deleted, needs recreation
    throw e; // Other error
  }
};

/**
 * Gets the channel name to use when creating a channel.
 * Prefers user-provided prename if available, otherwise uses default name.
 *
 * @param prename - User-provided prename from task configuration
 * @param defaultName - Default name to use if prename is not provided
 * @returns The channel name to use
 */
export const getChannelNameToCreate = (
  prename: string | undefined | null,
  defaultName: string,
): string => (primitive.isNonZero(prename) ? prename : defaultName);

/**
 * Validates that a list of channel names do not already exist in Synnax.
 * This prevents partial channel creation failures where some channels succeed
 * and others fail, leaving orphaned channels.
 *
 * @param client - Synnax client instance
 * @param channelNames - Array of channel names to validate
 * @throws Error if any of the channel names already exist
 * ```
 */
export const validateChannelNames = async (
  client: Synnax,
  channelNames: string[],
): Promise<void> => {
  if (channelNames.length === 0) return;

  const results = await Promise.all(
    channelNames.map(async (name) => {
      try {
        await client.channels.retrieve(name);
        return name; // Channel exists
      } catch (e) {
        if (!NotFoundError.matches(e)) throw e;
        return null; // Channel doesn't exist
      }
    }),
  );

  const existingChannels = results.filter((name): name is string => name !== null);

  if (existingChannels.length > 0)
    throw new Error(
      `Cannot configure task: the following channel${existingChannels.length > 1 ? "s" : ""} already exist${existingChannels.length === 1 ? "s" : ""}: ${existingChannels.join(", ")}. Rename the ${existingChannels.length > 1 ? "channels" : "channel"} or reconfigure the task with different settings.`,
    );
};

/**
 * Batch version of shouldCreateChannel that checks multiple keys in parallel.
 * Much faster than calling shouldCreateChannel in a loop for multiple channels.
 *
 * @param client - Synnax client instance
 * @param existingKeys - Array of channel keys to check
 * @returns Array of booleans indicating which channels need creation
 */
export const shouldCreateChannels = async (
  client: Synnax,
  existingKeys: Array<channel.Key | undefined | null>,
): Promise<boolean[]> =>
  await Promise.all(existingKeys.map((key) => shouldCreateChannel(client, key)));

/**
 * Helper for collecting index channel validation info.
 * Returns whether the index needs creation and its name if so.
 *
 * @param client - Synnax client instance
 * @param indexKey - Existing index channel key
 * @param indexName - Name to use if creating index
 * @returns Object with shouldCreate flag and nameToValidate (null if not creating)
 */
export const collectIndexChannelForValidation = async (
  client: Synnax,
  indexKey: channel.Key | undefined | null,
  indexName: string,
): Promise<{ shouldCreate: boolean; nameToValidate: string | null }> => {
  const shouldCreate = await shouldCreateChannel(client, indexKey);
  return {
    shouldCreate,
    nameToValidate: shouldCreate ? indexName : null,
  };
};

/**
 * Generic helper for collecting data channels that need creation (read tasks).
 * Checks all channels in parallel and returns which need creation + their names.
 *
 * @param client - Synnax client instance
 * @param channels - Array of channel configurations
 * @param getExistingKey - Function to extract existing key from channel config
 * @param getPrenameAndDefault - Function to get prename and default name for channel
 * @returns Object with channels to create and names to validate
 */
export const collectDataChannelsForValidation = async <TChannel>(
  client: Synnax,
  channels: TChannel[],
  getExistingKey: (ch: TChannel) => channel.Key | undefined | null,
  getPrenameAndDefault: (ch: TChannel) => {
    prename?: string | null;
    defaultName: string;
  },
): Promise<{ toCreate: TChannel[]; namesToValidate: string[] }> => {
  const shouldCreateFlags = await shouldCreateChannels(
    client,
    channels.map(getExistingKey),
  );

  const toCreate: TChannel[] = [];
  const namesToValidate: string[] = [];

  channels.forEach((ch, i) => {
    if (shouldCreateFlags[i]) {
      toCreate.push(ch);
      const { prename, defaultName } = getPrenameAndDefault(ch);
      namesToValidate.push(getChannelNameToCreate(prename, defaultName));
    }
  });

  return { toCreate, namesToValidate };
};

/**
 * Generic helper for collecting write channels (command + state pairs).
 * Handles the complexity of checking both command and state channels.
 *
 * @param client - Synnax client instance
 * @param channels - Array of channel configurations
 * @param getExistingPair - Function to extract existing command/state keys (null if neither exist)
 * @param getChannelNames - Function to generate all channel names (cmd, cmd_time, state)
 * @returns Object with channels to create for each type and all names to validate
 */
export const collectWriteChannelsForValidation = async <TChannel>(
  client: Synnax,
  channels: TChannel[],
  getExistingPair: (ch: TChannel) => {
    command: channel.Key | undefined | null;
    state: channel.Key | undefined | null;
  } | null,
  getChannelNames: (ch: TChannel) => {
    cmdPrename?: string | null;
    cmdDefault: string;
    cmdIndexDefault: string;
    statePrename?: string | null;
    stateDefault: string;
  },
): Promise<{
  commandsToCreate: TChannel[];
  statesToCreate: TChannel[];
  namesToValidate: string[];
}> => {
  const commandsToCreate: TChannel[] = [];
  const statesToCreate: TChannel[] = [];
  const namesToValidate: string[] = [];

  const keysToCheck: Array<channel.Key | undefined | null> = [];
  const checkTypes: Array<"command" | "state"> = [];

  for (const ch of channels) {
    const exPair = getExistingPair(ch);
    if (exPair == null) {
      // Neither exist - need to create both
      commandsToCreate.push(ch);
      statesToCreate.push(ch);
    } else {
      keysToCheck.push(exPair.state);
      checkTypes.push("state");
      keysToCheck.push(exPair.command);
      checkTypes.push("command");
    }
  }

  const shouldCreateFlags = await shouldCreateChannels(client, keysToCheck);

  // Process
  let checkIndex = 0;
  for (const ch of channels) {
    const exPair = getExistingPair(ch);
    if (exPair == null) continue;

    // State check
    if (shouldCreateFlags[checkIndex]) statesToCreate.push(ch);

    checkIndex++;

    // Command check
    if (shouldCreateFlags[checkIndex]) commandsToCreate.push(ch);

    checkIndex++;
  }

  // Collect
  for (const c of statesToCreate) {
    const names = getChannelNames(c);
    namesToValidate.push(
      getChannelNameToCreate(names.statePrename, names.stateDefault),
    );
  }

  for (const c of commandsToCreate) {
    const names = getChannelNames(c);
    namesToValidate.push(
      getChannelNameToCreate(
        primitive.isNonZero(names.cmdPrename) ? `${names.cmdPrename}_time` : undefined,
        names.cmdIndexDefault,
      ),
    );
    namesToValidate.push(getChannelNameToCreate(names.cmdPrename, names.cmdDefault));
  }

  return { commandsToCreate, statesToCreate, namesToValidate };
};
