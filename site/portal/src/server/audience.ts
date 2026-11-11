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
  /**
   * add puts the address of an account on the list. An address that Resend already
   * keeps holds on to its unsubscribe.
   */
  add: (email: string) => Promise<void>;
  /**
   * remove takes the address of a deleted account off the list. It deletes the contact,
   * except one that unsubscribed: Resend keeps that one outside every segment, so a
   * later add cannot undo the unsubscribe.
   */
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
  const retried = async <R extends Reply>(request: () => Promise<R>): Promise<R> => {
    for (;;) {
      const reply = await request();
      if (reply.error?.name !== "rate_limit_exceeded") return reply;
      const seconds = Number(reply.headers?.["retry-after"]);
      await sleep.sleep(
        TimeSpan.seconds(Number.isFinite(seconds) && seconds > 0 ? seconds : 1),
      );
    }
  };
  const unwrap = <R extends Reply>({ data, error }: R): NonNullable<R["data"]> => {
    if (error == null && data != null) return data;
    throw new Error(`audience: ${error?.message ?? "Resend sent no data"}`);
  };
  const call = async <R extends Reply>(
    request: () => Promise<R>,
  ): Promise<NonNullable<R["data"]>> => unwrap(await retried(request));
  const find = async (email: string): Promise<{ unsubscribed: boolean } | null> => {
    const reply = await retried(async () => await client.contacts.get({ email }));
    return reply.error?.name === "not_found" ? null : unwrap(reply);
  };
  const lookup = async (): Promise<Segments> => {
    const { data } = await call(
      async () => await client.segments.list({ limit: PAGE }),
    );
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
      if ((await find(email)) == null) {
        await call(
          async () =>
            await client.contacts.create({
              email,
              segments: [{ id: everyone }, { id: accounts }],
            }),
        );
        return;
      }
      for (const segmentId of [everyone, accounts])
        await call(
          async () => await client.contacts.segments.add({ email, segmentId }),
        );
    },
    remove: async (email) => {
      if ((await find(email))?.unsubscribed !== true) {
        await call(async () => await client.contacts.remove({ email }));
        return;
      }
      const { everyone, accounts } = await segments();
      for (const segmentId of [everyone, accounts])
        await call(
          async () => await client.contacts.segments.remove({ email, segmentId }),
        );
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

/** Contact is one address that a {@link memory} audience keeps. */
export interface Contact {
  unsubscribed: boolean;
  /** everyone is true when the address gets release notes. */
  everyone: boolean;
  /** account is true when the portal added the address for an account. */
  account: boolean;
}

/**
 * memory keeps the list in a map instead of Resend, with the same rules. It starts with
 * the addresses added by hand. For tests.
 */
export const memory = (
  hand: string[] = [],
): Audience & { contacts: Map<string, Contact> } => {
  const contacts = new Map<string, Contact>(
    hand.map((email) => [
      email,
      { unsubscribed: false, everyone: true, account: false },
    ]),
  );
  const members = (key: "everyone" | "account"): Set<string> =>
    new Set([...contacts].filter(([, c]) => c[key]).map(([email]) => email));
  return {
    contacts,
    everyone: async () => members("everyone"),
    accounts: async () => members("account"),
    add: async (email) => {
      const unsubscribed = contacts.get(email)?.unsubscribed ?? false;
      contacts.set(email, { unsubscribed, everyone: true, account: true });
    },
    remove: async (email) => {
      if (contacts.get(email)?.unsubscribed === true)
        contacts.set(email, { unsubscribed: true, everyone: false, account: false });
      else contacts.delete(email);
    },
  };
};
