// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sleep, TimeSpan } from "@synnaxlabs/x";
import { Resend } from "resend";

import { type Directory } from "@/server/directory";

/**
 * Audience is the release notes list. Resend keeps it, with the people added by hand
 * and the people who unsubscribed. Addresses are lowercase.
 */
export interface Audience {
  /** everyone returns every address on the list, unsubscribed ones included. */
  everyone: () => Promise<Set<string>>;
  /** accounts returns the addresses the portal added for its accounts. */
  accounts: () => Promise<Set<string>>;
  /** add puts the address of an account on the list. */
  add: (email: string) => Promise<void>;
  /** remove takes the address of a deleted account off the list. */
  remove: (email: string) => Promise<void>;
}

/** Synced counts the addresses that a sync added and removed. */
export interface Synced {
  added: number;
  removed: number;
}

/**
 * sync makes the accounts on the list match the accounts in `directory`. It adds only
 * addresses that are not on the list and removes only addresses that it added, so it
 * never removes a person added by hand or undoes an unsubscribe.
 */
export const sync = async (
  directory: Directory,
  audience: Audience,
): Promise<Synced> => {
  const [users, everyone, accounts] = await Promise.all([
    directory.users(),
    audience.everyone(),
    audience.accounts(),
  ]);
  const current = new Set(
    users.map(({ email }) => email.toLowerCase()).filter((email) => email !== ""),
  );
  const added = [...current].filter((email) => !everyone.has(email));
  const removed = [...accounts].filter((email) => !current.has(email));
  for (const email of added) await audience.add(email);
  for (const email of removed) await audience.remove(email);
  return { added: added.length, removed: removed.length };
};

/**
 * SEGMENTS names the Resend segments of the list. Release notes go to `everyone`;
 * `accounts` holds the addresses the portal added, so sync never removes a person added
 * by hand.
 */
const SEGMENTS = { everyone: "Release notes", accounts: "Portal accounts" };

type Segments = Record<keyof typeof SEGMENTS, string>;

/** Reply is the shape every Resend SDK call resolves to. */
interface Reply {
  data: unknown;
  error: { message: string; name: string } | null;
  headers: Record<string, string> | null;
}

const PAGE = 100;

/** resend keeps the list in Resend. It throws when a segment in SEGMENTS is missing. */
export const resend = (apiKey: string): Audience => {
  const client = new Resend(apiKey);
  // Resend answers 429 when a team passes its request rate.
  const call = async <R extends Reply>(
    request: () => Promise<R>,
  ): Promise<NonNullable<R["data"]>> => {
    for (;;) {
      const { data, error, headers } = await request();
      if (error == null && data != null) return data;
      if (error?.name !== "rate_limit_exceeded")
        throw new Error(`audience: ${error?.message ?? "Resend sent no data"}`);
      await sleep.sleep(TimeSpan.seconds(Number(headers?.["retry-after"] ?? 1)));
    }
  };
  const lookup = async (): Promise<Segments> => {
    const { data } = await call(async () => await client.segments.list());
    const id = (name: string): string => {
      const segment = data.find((s) => s.name === name);
      if (segment == null) throw new Error(`audience: no Resend segment ${name}`);
      return segment.id;
    };
    return { everyone: id(SEGMENTS.everyone), accounts: id(SEGMENTS.accounts) };
  };
  let ids: Promise<Segments> | undefined;
  const segments = async (): Promise<Segments> => await (ids ??= lookup());
  const members = async (segment: keyof Segments): Promise<Set<string>> => {
    const segmentId = (await segments())[segment];
    const emails = new Set<string>();
    let after: string | undefined;
    for (;;) {
      const page = await call(
        async () => await client.contacts.list({ segmentId, limit: PAGE, after }),
      );
      for (const { email } of page.data) emails.add(email.toLowerCase());
      after = page.data.at(-1)?.id;
      if (!page.has_more || after == null) return emails;
    }
  };
  return {
    everyone: async () => await members("everyone"),
    accounts: async () => await members("accounts"),
    add: async (email) => {
      const { everyone, accounts } = await segments();
      await call(
        async () =>
          await client.contacts.create({
            email,
            segments: [{ id: everyone }, { id: accounts }],
          }),
      );
    },
    remove: async (email) => {
      await call(async () => await client.contacts.remove({ email }));
    },
  };
};

/** log prints changes instead of making them, so previews never touch the list. */
export const log = (): Audience => ({
  everyone: async () => new Set(),
  accounts: async () => new Set(),
  add: async (email) => console.info("audience add skipped", { email }),
  remove: async (email) => console.info("audience remove skipped", { email }),
});

/**
 * memory keeps the list in sets instead of Resend. `everyone` starts with the addresses
 * added by hand. For tests.
 */
export const memory = (
  hand: string[] = [],
): Audience & { list: Set<string>; synced: Set<string> } => {
  const list = new Set(hand);
  const synced = new Set<string>();
  return {
    list,
    synced,
    everyone: async () => new Set(list),
    accounts: async () => new Set(synced),
    add: async (email) => {
      list.add(email);
      synced.add(email);
    },
    remove: async (email) => {
      list.delete(email);
      synced.delete(email);
    },
  };
};
