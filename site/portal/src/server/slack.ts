// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Action, ACTIONS, replayURL, type Visit } from "@/server/analytics";
import { type License, type Organization } from "@/server/db/schema";
import { type Account } from "@/server/directory";
import { type Denial } from "@/server/license/deny";
import { type Fact } from "@/server/mail";
import { DAY_MS, dayOf, MONTH } from "@/server/usage";
import { type Arrival, type Summary } from "@/server/user";
import { userPath } from "@/shell";

export interface Link {
  label: string;
  /** href is a path on the portal, such as `/licenses/<key>`, or a full URL. */
  href: string;
}

/** Note is one message to the staff channel. */
export interface Note {
  /** title says what happened in a few words. */
  title: string;
  facts: Fact[];
  links: Link[];
}

/** Slack posts notes to the staff channel. */
export interface Slack {
  /** post never throws, so a note cannot fail the request that made it. */
  post: (note: Note) => Promise<void>;
}

/** escape keeps text from reading as markup: Slack treats &, <, and > as markup. */
const escape = (text: string): string =>
  text.replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;");

const resolve = (href: string, origin: string): string =>
  href.startsWith("/") ? `${origin}${href}` : href;

/** render writes a note in Slack's markup, with its portal links on `origin`. */
const render = ({ title, facts, links }: Note, origin: string): string =>
  [
    `*${escape(title)}*`,
    ...facts.map(({ label, value }) => `${escape(label)}: ${escape(value)}`),
    links
      .map(({ label, href }) => `<${resolve(href, origin)}|${escape(label)}>`)
      .join(" · "),
  ].join("\n");

const TIMEOUT_MS = 5000;

export interface WebhookConfig {
  url: string;
  /** origin is the portal's URL, which the links in a note point to. */
  origin: string;
}

/** webhook posts through a Slack incoming webhook and logs a post that fails. */
export const webhook = ({ url, origin }: WebhookConfig): Slack => ({
  post: async (note) => {
    try {
      const res = await fetch(url, {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ text: render(note, origin) }),
        signal: AbortSignal.timeout(TIMEOUT_MS),
      });
      if (!res.ok)
        console.error("slack post failed", {
          title: note.title,
          status: res.status,
          body: await res.text(),
        });
    } catch (err) {
      console.error("slack post failed", { title: note.title }, err);
    }
  },
});

/** log prints notes instead of posting them, so previews never post. */
export const log = (): Slack => ({
  post: async ({ title }) => console.info("slack note not posted", { title }),
});

/**
 * announce posts the note that `make` returns, if any. Like {@link Slack.post}, it
 * never throws: a failure to make the note is logged, so it cannot fail the request.
 */
export const announce = async (
  slack: Slack,
  make: () => Promise<Note | null>,
): Promise<void> => {
  let note: Note | null;
  try {
    note = await make();
  } catch (err) {
    console.error("slack note failed", err);
    return;
  }
  if (note != null) await slack.post(note);
};

/** memory collects notes instead of posting them. For tests. */
export const memory = (): Slack & { posted: Note[] } => {
  const posted: Note[] = [];
  return { posted, post: async (note) => void posted.push(note) };
};

const openUser = (userID: string): Link => ({
  label: "Open the user",
  href: userPath(userID),
});

const person = (account: Pick<Account, "name" | "email">): Fact[] => [
  { label: "Name", value: account.name },
  { label: "Email", value: account.email },
];

/** joined tells staff that someone created an account. */
export const joined = (account: Omit<Account, "joinedAt">): Note => ({
  title: "New account",
  facts: person(account),
  links: [openUser(account.userID)],
});

/** firstLogin tells staff about the first login of a user to Synnax Desktop. */
export const firstLogin = (
  account: Omit<Account, "joinedAt">,
  machine: string,
): Note => ({
  title: "First Desktop login",
  facts: [...person(account), { label: "Machine", value: machine }],
  links: [openUser(account.userID)],
});

const DENIALS: Record<Denial, string> = {
  revoked: "The license is revoked",
  expired: "The license expired",
  no_seats: "Every seat is taken",
};

export interface RefusalArgs {
  organization: Organization;
  license: License;
  reason: Denial;
  /** actor is who tried to activate. */
  actor: Pick<Account, "name" | "email">;
}

/** refused tells staff that a machine could not activate an enterprise license. */
export const refused = ({
  organization,
  license,
  reason,
  actor,
}: RefusalArgs): Note => ({
  title: "Activation refused",
  facts: [
    { label: "Organization", value: organization.name },
    { label: "License", value: license.label },
    { label: "Reason", value: DENIALS[reason] },
    { label: "By", value: `${actor.name} (${actor.email})` },
  ],
  links: [{ label: "Open the license", href: `/licenses/${license.key}` }],
});

/** digest reports a day of Synnax Desktop use. */
export const digest = ({ day, active, accounts, firstLogins }: Summary): Note => ({
  title: `Synnax Desktop on ${day.toISOString().slice(0, 10)}`,
  facts: [
    { label: "Daily active users", value: String(active.day) },
    { label: "Weekly active users", value: String(active.week) },
    { label: "Monthly active users", value: String(active.month) },
    { label: "New accounts", value: String(accounts) },
    { label: "First Desktop logins", value: String(firstLogins) },
  ],
  links: [{ label: "Open the users", href: "/admin/users" }],
});

/** clock reads a time as hours and minutes in UTC. */
const clock = (t: Date): string => t.toISOString().slice(11, 16);

/** ago says how many UTC days before `now` the day of `then` was. */
const ago = (then: Date, now: Date): string => {
  const days = Math.round((dayOf(now).getTime() - dayOf(then).getTime()) / DAY_MS);
  return days === 1 ? "Yesterday" : `${days} days ago`;
};

/** active tells staff that a user started using Synnax Desktop for the day. */
export const active = ({ account, machine, at, previous, days }: Arrival): Note => ({
  title: `${account.name} is active today`,
  facts: [
    { label: "Email", value: account.email },
    { label: "Machine", value: machine },
    { label: "Time", value: `${clock(at)} UTC` },
    { label: "Last active", value: previous == null ? "Never" : ago(previous, at) },
    { label: "Active days", value: `${days} of the last ${MONTH}` },
  ],
  links: [openUser(account.userID)],
});

const MINUTE_MS = 60 * 1000;

/** span reads a length of time in hours and minutes. */
const span = (ms: number): string => {
  const minutes = Math.floor(ms / MINUTE_MS);
  if (minutes < 1) return "under a minute";
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  if (h === 0) return `${m} min`;
  return m === 0 ? `${h} h` : `${h} h ${m} min`;
};

/** ACTION_LABELS name what each key action made, and what it did to it. */
const ACTION_LABELS: Record<Action, { noun: string; verb: string }> = {
  channel_created: { noun: "channel", verb: "created" },
  device_connected: { noun: "device", verb: "connected" },
  device_configured: { noun: "device", verb: "configured" },
  task_started: { noun: "task", verb: "started" },
  automation_deployed: { noun: "automation", verb: "deployed" },
  plot_created: { noun: "line plot", verb: "created" },
  schematic_created: { noun: "schematic", verb: "created" },
  range_created: { noun: "range", verb: "created" },
};

const actionsOf = ({ actions }: Visit): string => {
  const done = ACTIONS.flatMap((a) => {
    const n = actions[a];
    if (n == null) return [];
    const { noun, verb } = ACTION_LABELS[a];
    return [`${n} ${n === 1 ? noun : `${noun}s`} ${verb}`];
  });
  return done.length === 0 ? "None" : done.join(", ");
};

/**
 * recap tells staff what a user did in one Synnax Desktop visit. `account` is null
 * when the visit has no logged-in user that Clerk knows.
 */
export const recap = (visit: Visit, account: Account | null): Note => {
  const who = account?.name ?? visit.email ?? "Someone who is not logged in";
  const duration = visit.end.getTime() - visit.start.getTime();
  const email = account?.email ?? visit.email;
  const facts: Fact[] = [];
  if (email != null && email !== who) facts.push({ label: "Email", value: email });
  facts.push({
    label: "Time",
    value: `${clock(visit.start)} to ${clock(visit.end)} UTC`,
  });
  if (visit.version != null)
    facts.push({
      label: "App",
      value: visit.os == null ? visit.version : `${visit.version} on ${visit.os}`,
    });
  facts.push({ label: "Actions", value: actionsOf(visit) });
  facts.push({
    label: "Screens",
    value: visit.screens.length === 0 ? "None" : visit.screens.join(", "),
  });
  const links = [{ label: "Watch the replay", href: replayURL(visit.replay) }];
  if (account != null) links.unshift(openUser(account.userID));
  return {
    title: `${who} used Synnax Desktop for ${span(duration)}`,
    facts,
    links,
  };
};
