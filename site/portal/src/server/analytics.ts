// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

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

/** Session is one PostHog session of Synnax Desktop. */
export interface Session {
  id: string;
  /** distinctID is who PostHog saw last: the Clerk user ID once the user logs in. */
  distinctID: string;
  /** email is the person's email, or null before they log in. */
  email: string | null;
  start: Date;
  end: Date;
  /** version is the app version, or null when the app opened in an earlier session. */
  version: string | null;
  os: string | null;
  /** screens are the tab types the user looked at, in no order. */
  screens: string[];
  /** actions counts each key action, leaving out those with none. */
  actions: Partial<Record<Action, number>>;
}

/** Analytics reads what users did in Synnax Desktop, for users who allow analytics. */
export interface Analytics {
  /**
   * sessions returns the production sessions whose last event falls in [from, to),
   * oldest first.
   */
  sessions: (from: Date, to: Date) => Promise<Session[]>;
}

/** clickhouse writes a time as a UTC literal that HogQL compares against. */
const clickhouse = (t: Date): string =>
  `toDateTime('${t.toISOString().slice(0, 19).replace("T", " ")}', 'UTC')`;

/**
 * query selects each session with its last event in [from, to). A session lasts at most
 * a day, so events from a day before `from` cover every session that ends in the range.
 */
const query = (from: Date, to: Date): string => `
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
WHERE timestamp >= ${clickhouse(from)} - INTERVAL 1 DAY
  AND properties.environment = 'production'
  AND $session_id != ''
GROUP BY $session_id
HAVING max(timestamp) >= ${clickhouse(from)} AND max(timestamp) < ${clickhouse(to)}
ORDER BY min(timestamp)`;

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

/** posthog reads sessions through PostHog's query API with a personal API key. */
export const posthog = (apiKey: string): Analytics => ({
  sessions: async (from, to) => {
    const res = await fetch(`${HOST}/api/projects/${PROJECT}/query/`, {
      method: "POST",
      headers: {
        authorization: `Bearer ${apiKey}`,
        "content-type": "application/json",
      },
      body: JSON.stringify({ query: { kind: "HogQLQuery", query: query(from, to) } }),
      signal: AbortSignal.timeout(TIMEOUT_MS),
    });
    if (!res.ok)
      throw new Error(`posthog: query failed with ${res.status}: ${await res.text()}`);
    return responseZ.parse(await res.json()).results;
  },
});

/** unconfigured is the analytics of a portal with no PostHog key. Every read throws. */
export const unconfigured = (): Analytics => ({
  sessions: async () => {
    throw new Error("POSTHOG_API_KEY is not set");
  },
});

/** memory serves the sessions in `recorded` instead of PostHog. For tests. */
export const memory = (): Analytics & { recorded: Session[] } => {
  const recorded: Session[] = [];
  return {
    recorded,
    sessions: async (from, to) => recorded.filter((s) => s.end >= from && s.end < to),
  };
};
