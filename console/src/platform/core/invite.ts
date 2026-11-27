// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type ConnectionParams,
  detectConnection,
} from "@/platform/core/detectConnection";
import type { Session } from "@/session";

type Address = Pick<ConnectionParams, "host" | "port" | "secure">;

/** The start of a link that opens the desktop Console on a Core with an invite. */
export const DESKTOP_INVITE_PREFIX = "synnax://invite";

const FRAGMENT_KEY = "invite";

const fragment = (code: string): string =>
  `#${new URLSearchParams({ [FRAGMENT_KEY]: code }).toString()}`;

// an IPv6 address needs brackets before its port
const urlHost = (host: string): string =>
  host.includes(":") && !host.startsWith("[") ? `[${host}]` : host;

/** Links that open a Console on the Core with an invite code filled in. */
export interface InviteLinks {
  /** Opens the web Console that the Core serves. */
  browser: string;
  /** Opens the desktop Console on the saved Core at the link's address. */
  desktop: string;
}

/**
 * Returns the links that carry the invite code. The code is in the fragment of each,
 * which a browser never sends to a server.
 */
export const createInviteLinks = (
  { host, port, secure }: Address,
  code: string,
): InviteLinks => {
  const address = new URLSearchParams({ host, port: port.toString() });
  if (secure) address.set("secure", "true");
  return {
    browser: `${secure ? "https" : "http"}://${urlHost(host)}:${port}/${fragment(code)}`,
    desktop: `${DESKTOP_INVITE_PREFIX}?${address.toString()}${fragment(code)}`,
  };
};

const parseInviteFragment = (hash: string): string | undefined => {
  const code = new URLSearchParams(hash.replace(/^#/, "")).get(FRAGMENT_KEY)?.trim();
  return code == null || code === "" ? undefined : code;
};

/** A desktop invite link, read. */
export interface DesktopInvite extends Address {
  code: string;
}

/**
 * Reads a link from {@link createInviteLinks}.
 * @throws {Error} when the link has no Core address or no invite code.
 */
export const parseDesktopInvite = (link: string): DesktopInvite => {
  const url = new URL(link);
  const host = url.searchParams.get("host");
  const port = Number(url.searchParams.get("port"));
  const code = parseInviteFragment(url.hash);
  if (host == null || host === "" || !Number.isInteger(port) || port <= 0)
    throw new Error("The invite link has no Core address.");
  if (code == null) throw new Error("The invite link has no invite code.");
  return { host, port, secure: url.searchParams.get("secure") === "true", code };
};

/**
 * Takes the invite code from the URL fragment of a page that a Core serves, and removes
 * the fragment from the address bar and the history entry. Call it once, before the
 * Console renders.
 * @returns the invite for the serving Core, or null when the page holds none.
 */
export const takeBrowserInvite = (): Session.Link.Invite | null => {
  const code = parseInviteFragment(window.location.hash);
  if (code == null) return null;
  const { pathname, search } = window.location;
  window.history.replaceState(window.history.state, "", `${pathname}${search}`);
  const core = detectConnection();
  return core == null ? null : { core: core.key, code };
};
