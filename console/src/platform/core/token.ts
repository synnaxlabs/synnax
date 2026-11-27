// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { invoke } from "@tauri-apps/api/core";

import { Session } from "@/session";

type Address = Pick<Session.Core.Core, "host" | "port" | "secure" | "username">;

type Target = Address & Pick<Session.Core.Core, "key">;

/**
 * The scheme, user, and address of a Core. The scheme is part of it, so a token saved
 * for TLS never goes out in plain text.
 */
export const addressOf = ({ host, port, secure, username }: Address): string =>
  `${secure ? "https" : "http"}://${encodeURIComponent(username)}@${host}:${port}`;

/**
 * The keychain account a saved Core's session token is stored under. It holds the key
 * of the saved Core, so two saved Cores never resume each other's session.
 */
export const accountOf = (core: Target): string => `${addressOf(core)}#${core.key}`;

// The dev server has no keychain, so its sessions last as long as the tab.
const storageKey = (core: Target): string => `synnax-session-token:${accountOf(core)}`;

/** Reports whether this tab holds a session token for the Core, on the dev server. */
export const tabHoldsToken = (core: Target): boolean =>
  sessionStorage.getItem(storageKey(core)) != null;

/**
 * Reads the session token saved for the Core: from the keychain on desktop and from
 * the tab on the dev server. Resolves undefined when none is saved, and always on a
 * page a Core serves.
 *
 * @throws if the keychain cannot be read.
 */
export const readToken = async (core: Target): Promise<string | undefined> => {
  if (!Session.Runtime.CORE_SELECTABLE) return undefined;
  if (Session.Runtime.ENGINE !== "tauri")
    return sessionStorage.getItem(storageKey(core)) ?? undefined;
  const token = await invoke<string | null>("keychain_get", {
    account: accountOf(core),
  });
  return token ?? undefined;
};

/**
 * Saves the Core's session token. A no-op on a page a Core serves.
 *
 * @throws if the keychain cannot be written.
 */
export const writeToken = async (core: Target, token: string): Promise<void> => {
  if (!Session.Runtime.CORE_SELECTABLE) return;
  if (Session.Runtime.ENGINE !== "tauri") {
    sessionStorage.setItem(storageKey(core), token);
    return;
  }
  await invoke("keychain_set", { account: accountOf(core), secret: token });
};

/**
 * Removes the Core's saved session token. A no-op on a page a Core serves.
 *
 * @param expected - Removes the token only while it is still this one, so a token
 * another window saved since stays.
 * @throws if the keychain cannot be written.
 */
export const deleteToken = async (core: Target, expected?: string): Promise<void> => {
  if (!Session.Runtime.CORE_SELECTABLE) return;
  if (Session.Runtime.ENGINE !== "tauri") {
    const key = storageKey(core);
    if (expected == null || sessionStorage.getItem(key) === expected)
      sessionStorage.removeItem(key);
    return;
  }
  await invoke("keychain_delete", { account: accountOf(core), expected });
};

/**
 * Moves the session token the keychain holds for one Core to another, such as the same
 * Core under a new username. It writes the new entry before it deletes the old one. A
 * no-op on a page a Core serves, and when no token is saved.
 *
 * @throws if the keychain cannot be read or written.
 */
export const moveToken = async (from: Target, to: Target): Promise<void> => {
  const token = await readToken(from);
  if (token == null) return;
  await writeToken(to, token);
  await deleteToken(from, token);
};
