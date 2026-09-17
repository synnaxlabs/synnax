// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIContext } from "astro";

import { open, type Portal } from "@/portal/portal";
import { HTTPError } from "@/server/errors";
import { type Session } from "@/server/session";

export interface Loaded {
  portal: Portal;
  session: Session;
}

/**
 * load opens the portal for a signed-in page. A signed-out visitor gets a redirect to
 * sign in that returns them here afterwards.
 */
export const load = async (context: APIContext): Promise<Loaded | Response> => {
  const portal = open(context);
  try {
    return { portal, session: await portal.session() };
  } catch (err) {
    if (!(err instanceof HTTPError) || err.status !== 401)
      throw err instanceof Error ? err : new Error(String(err));
    const back = context.url.pathname + context.url.search;
    return context.redirect(`/sign-in?redirect_url=${encodeURIComponent(back)}`, 303);
  }
};

/** failure reads the status and message a page shows for an error it caught. */
export const failure = (err: unknown): { status: number; message: string } => {
  if (err instanceof HTTPError) return { status: err.status, message: err.message };
  console.error(err);
  return { status: 500, message: "Something went wrong. Try again." };
};
