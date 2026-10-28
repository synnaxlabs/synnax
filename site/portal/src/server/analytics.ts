// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { DAY_MS } from "@/server/usage";

/** PROJECT is the PostHog project that Synnax Desktop reports to. */
const PROJECT = 621680;

const HOST = "https://us.posthog.com";

/**
 * personURL opens a user's person in PostHog. Synnax Desktop identifies the person by
 * the user's Clerk user ID.
 */
export const personURL = (userID: string): string =>
  `${HOST}/project/${PROJECT}/person/${encodeURIComponent(userID)}`;

/** replayURL opens the recording of a session in PostHog. */
export const replayURL = (sessionID: string): string =>
  `${HOST}/project/${PROJECT}/replay/${encodeURIComponent(sessionID)}`;

/** IDLE_MS is how long PostHog waits without an event before a session ends. */
export const IDLE_MS = 30 * 60 * 1000;

/** ACTIONS are the events Synnax Desktop sends for the key actions of a user. */
export const ACTIONS = [
  "channel_created",
  "device_connected",
  "device_configured",
  "task_started",
  "automation_deployed",
  "plot_created",
  "schematic_created",
  "range_created",
] as const;

export type Action = (typeof ACTIONS)[number];

/**
 * Visit is one stretch of a person's Synnax Desktop use. PostHog starts a session in
 * each window and after each reset, so a visit joins the sessions of one person that
 * are less than {@link IDLE_MS} apart.
 */
export interface Visit {
  /** distinctID is who PostHog saw last: the Clerk user ID once the user logs in. */
  distinctID: string;
  /** email is the person's email, or null before they log in. */
  email: string | null;
  start: Date;
  end: Date;
  /** version is the app version, or null when the app opened before the visit. */
  version: string | null;
  os: string | null;
  /** screens are the tab types the user looked at, in no order. */
  screens: string[];
  /** actions counts each key action, leaving out those with none. */
  actions: Partial<Record<Action, number>>;
  /** replay is the ID of the longest session, which records the main window. */
  replay: string;
}

/** Session is one PostHog session of Synnax Desktop. */
export interface Session extends Omit<Visit, "replay"> {
  id: string;
}

/** Analytics reads what users did in Synnax Desktop, for users who allow analytics. */
export interface Analytics {
  /**
   * visits returns the production visits whose last event falls in [from, to), oldest
   * first. A visit can go on until {@link IDLE_MS} after its last event, so `to` must
   * be at least that far in the past.
   */
  visits: (from: Date, to: Date) => Promise<Visit[]>;
}

/** LOOKBACK_MS is how long before a visit ends that its sessions are read from. */
const LOOKBACK_MS = DAY_MS;

const lookback = (from: Date): Date => new Date(from.getTime() - LOOKBACK_MS);

const span = ({ start, end }: Session): number => end.getTime() - start.getTime();

/** Group is the sessions of one visit, oldest first, and the end of the newest. */
interface Group {
  sessions: Session[];
  end: number;
}

const join = ({ sessions, end }: Group): Visit => {
  const actions: Partial<Record<Action, number>> = {};
  for (const s of sessions)
    for (const a of ACTIONS) {
      const n = s.actions[a];
      if (n != null) actions[a] = (actions[a] ?? 0) + n;
    }
  return {
    distinctID: sessions[0].distinctID,
    email: sessions.find((s) => s.email != null)?.email ?? null,
    start: sessions[0].start,
    end: new Date(end),
    version: sessions.find((s) => s.version != null)?.version ?? null,
    os: sessions.find((s) => s.os != null)?.os ?? null,
    screens: [...new Set(sessions.flatMap((s) => s.screens))],
    actions,
    replay: sessions.reduce((a, b) => (span(b) > span(a) ? b : a)).id,
  };
};

/** visitsIn joins each person's sessions into visits, and keeps those in [from, to). */
const visitsIn = (sessions: Session[], from: Date, to: Date): Visit[] => {
  const sorted = [...sessions].sort(
    (a, b) =>
      a.distinctID.localeCompare(b.distinctID) || a.start.getTime() - b.start.getTime(),
  );
  const groups: Group[] = [];
  for (const s of sorted) {
    const group = groups.at(-1);
    if (
      group?.sessions[0].distinctID === s.distinctID &&
      s.start.getTime() < group.end + IDLE_MS
    ) {
      group.sessions.push(s);
      group.end = Math.max(group.end, s.end.getTime());
    } else groups.push({ sessions: [s], end: s.end.getTime() });
  }
  return groups
    .map(join)
    .filter((v) => v.end >= from && v.end < to)
    .sort((a, b) => a.start.getTime() - b.start.getTime());
};

/** clickhouse writes a time as a UTC literal that HogQL compares against. */
const clickhouse = (t: Date): string =>
  `toDateTime('${t.toISOString().slice(0, 19).replace("T", " ")}', 'UTC')`;

/** query selects each session with its events since `since`. */
const query = (since: Date): string => `
SELECT
  $session_id,
  argMax(distinct_id, timestamp),
  any(person.properties.email),
  toUnixTimestamp(min(timestamp)),
  toUnixTimestamp(max(timestamp)),
  anyIf(properties.version, event = 'app_opened'),
  anyIf(properties.os, event = 'app_opened'),
  groupUniqArrayIf(properties.$pathname, event = '$pageview'),
  ${ACTIONS.map((a) => `countIf(event = '${a}')`).join(",\n  ")}
FROM events
WHERE timestamp >= ${clickhouse(since)}
  AND properties.environment = 'production'
  AND $session_id != ''
GROUP BY $session_id`;

const optional = z
  .string()
  .nullable()
  .transform((s) => (s === "" ? null : s));

const seconds = z.number().transform((s) => new Date(s * 1000));

const rowZ = z
  .tuple(
    [
      z.string(),
      z.string(),
      optional,
      seconds,
      seconds,
      optional,
      optional,
      z.array(z.string().nullable()),
    ],
    z.number(),
  )
  .transform(([id, distinctID, email, start, end, version, os, paths, ...counts]) => ({
    id,
    distinctID,
    email,
    start,
    end,
    version,
    os,
    screens: paths.flatMap((p) =>
      p == null || p === "" ? [] : [p.replace(/^\//, "")],
    ),
    actions: Object.fromEntries(
      ACTIONS.flatMap((a, i) => (counts[i] > 0 ? [[a, counts[i]]] : [])),
    ),
  }));

const responseZ = z.object({ results: z.array(rowZ) });

const TIMEOUT_MS = 30_000;

/** posthog reads visits through PostHog's query API with a personal API key. */
export const posthog = (apiKey: string): Analytics => ({
  visits: async (from, to) => {
    const res = await fetch(`${HOST}/api/projects/${PROJECT}/query/`, {
      method: "POST",
      headers: {
        authorization: `Bearer ${apiKey}`,
        "content-type": "application/json",
      },
      body: JSON.stringify({
        query: { kind: "HogQLQuery", query: query(lookback(from)) },
      }),
      signal: AbortSignal.timeout(TIMEOUT_MS),
    });
    if (!res.ok)
      throw new Error(`posthog: query failed with ${res.status}: ${await res.text()}`);
    return visitsIn(responseZ.parse(await res.json()).results, from, to);
  },
});

/** unconfigured is the analytics of a portal with no PostHog key. Every read throws. */
export const unconfigured = (): Analytics => ({
  visits: async () => {
    throw new Error("POSTHOG_API_KEY is not set");
  },
});

/** memory serves visits of the sessions in `recorded` instead of PostHog. For tests. */
export const memory = (): Analytics & { recorded: Session[] } => {
  const recorded: Session[] = [];
  return {
    recorded,
    visits: async (from, to) =>
      visitsIn(
        recorded.filter((s) => s.end >= lookback(from)),
        from,
        to,
      ),
  };
};
