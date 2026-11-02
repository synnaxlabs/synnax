// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { auth, InvalidTokenError, type Synnax } from "@synnaxlabs/client";
import { errors } from "@synnaxlabs/x";

import { retrieveAPIKey, saveAPIKey } from "@/feature/embedded/supervisor";

const NAME = "Scripts on this computer";

const recordKey = (token: string): auth.APIKeyKey | null => {
  try {
    return auth.apiKeyID(token);
  } catch (e) {
    // The file sits in the home directory, where a person can edit it.
    if (InvalidTokenError.matches(e)) return null;
    throw errors.fromUnknown(e);
  }
};

const known = async (client: Synnax, token: string): Promise<boolean> => {
  const key = recordKey(token);
  if (key == null) return false;
  const keys = await client.auth.retrieveKeys();
  return keys.some((k) => k.key === key);
};

/** @returns The new key, or null when another window changed the saved key first. */
const create = async (
  client: Synnax,
  replaces: string | null,
): Promise<string | null> => {
  const { token, key } = await client.auth.createKey({ name: NAME });
  let saved: boolean;
  try {
    saved = await saveAPIKey(token, replaces);
  } catch (e) {
    await client.auth.deleteKeys(key.key).catch(console.error);
    throw errors.fromUnknown(e);
  }
  if (saved) return token;
  await client.auth.deleteKeys(key.key);
  return null;
};

/**
 * Makes sure that the saved API key belongs to the caller and that the saved address is
 * that of the running Core. Creates and saves a new key when the saved one is missing
 * or the Core does not know it.
 * @returns The saved API key.
 */
export const ensure = async (client: Synnax): Promise<string> => {
  const saved = await retrieveAPIKey();
  if (saved != null && (await known(client, saved))) {
    if (await saveAPIKey(saved, saved)) return saved;
  } else {
    const token = await create(client, saved);
    if (token != null) return token;
  }
  // Another window changed the saved key between the read and the save.
  return await ensure(client);
};

/**
 * Deletes the saved API key, which ends every session it opened, then saves a new one.
 * @returns The new API key.
 */
export const replace = async (client: Synnax): Promise<string> => {
  const saved = await retrieveAPIKey();
  const old = saved == null ? null : recordKey(saved);
  // The delete comes first, so a reset that fails never leaves the old key working.
  if (old != null) await client.auth.deleteKeys(old);
  // Another window reset the key first, so its new key is the one to use.
  return (await create(client, saved)) ?? (await ensure(client));
};

/** @returns The number of the given sessions that the given API key opened. */
export const countSessions = (sessions: auth.Session[], token: string): number => {
  const key = recordKey(token);
  return sessions.filter((s) => s.credential === key).length;
};
