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

const create = async (client: Synnax): Promise<string> => {
  const { token, key } = await client.auth.createKey({ name: NAME });
  try {
    await saveAPIKey(token);
  } catch (e) {
    await client.auth.deleteKeys(key.key).catch(console.error);
    throw errors.fromUnknown(e);
  }
  return token;
};

/**
 * Makes sure that the saved API key belongs to the caller and that the saved address is
 * that of the running Core. Creates and saves a new key when the saved one is missing
 * or the Core does not know it.
 * @returns The saved API key.
 */
export const ensure = async (client: Synnax): Promise<string> => {
  const saved = await retrieveAPIKey();
  const key = saved == null ? null : recordKey(saved);
  if (saved == null || key == null) return await create(client);
  const keys = await client.auth.retrieveKeys();
  if (!keys.some((k) => k.key === key)) return await create(client);
  await saveAPIKey(saved);
  return saved;
};

/**
 * Saves a new API key and deletes the old one, which ends every session the old key
 * opened.
 * @returns The new API key.
 */
export const replace = async (client: Synnax): Promise<string> => {
  const saved = await retrieveAPIKey();
  const token = await create(client);
  const old = saved == null ? null : recordKey(saved);
  if (old != null) await client.auth.deleteKeys(old);
  return token;
};

/** @returns The number of sessions that the given API key has open. */
export const countSessions = async (client: Synnax, token: string): Promise<number> => {
  const key = recordKey(token);
  const sessions = await client.auth.retrieveSessions();
  return sessions.filter((s) => s.credential === key).length;
};
