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

/**
 * Reads the session token the keychain holds for the Core. Resolves undefined when
 * none is stored, and always on the web, which has no keychain.
 *
 * @throws if the keychain cannot be read.
 */
export const readToken = async (core: Target): Promise<string | undefined> => {
  if (Session.Runtime.ENGINE !== "tauri") return undefined;
  const token = await invoke<string | null>("keychain_get", {
    account: accountOf(core),
  });
  return token ?? undefined;
};

/**
 * Stores the Core's session token in the keychain. A no-op on the web.
 *
 * @throws if the keychain cannot be written.
 */
export const writeToken = async (core: Target, token: string): Promise<void> => {
  if (Session.Runtime.ENGINE !== "tauri") return;
  await invoke("keychain_set", { account: accountOf(core), secret: token });
};

/**
 * Removes the Core's session token from the keychain. A no-op on the web.
 *
 * @throws if the keychain cannot be written.
 */
export const deleteToken = async (core: Target): Promise<void> => {
  if (Session.Runtime.ENGINE !== "tauri") return;
  await invoke("keychain_delete", { account: accountOf(core) });
};
