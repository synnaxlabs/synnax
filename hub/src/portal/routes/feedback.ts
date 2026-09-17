// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";
import { z } from "zod";

import { open } from "@/portal/portal";
import { badRequest, HTTPError, toResponse } from "@/server/errors";
import { ensurePersonal } from "@/server/organization";
import { checkFeedback } from "@/server/ratelimit";
import { open as openThread } from "@/server/support/thread";

const bodyZ = z.object({
  name: z.string().trim().max(200).default(""),
  email: z.union([z.literal(""), z.email()]).default(""),
  description: z.string().trim().min(1).max(50_000),
  page: z.string().trim().max(500).default("/"),
});

/** ANONYMOUS names a submission that gave no name. */
export const ANONYMOUS = "Anonymous visitor";

/**
 * POST turns a docs feedback submission into a thread. A signed-in visitor files it
 * under their personal organization and can follow it in the portal; anyone else is
 * reached at the email they gave, if any.
 */
export const POST: APIRoute = async (context) => {
  try {
    const parsed = bodyZ.safeParse(await context.request.json());
    if (!parsed.success) throw badRequest("Enter a message and a valid email");
    const { name, email, description, page } = parsed.data;
    const portal = open(context);
    let session: Awaited<ReturnType<typeof portal.session>> | null = null;
    try {
      session = await portal.session();
    } catch (err) {
      if (!(err instanceof HTTPError) || err.status !== 401)
        throw err instanceof Error ? err : new Error(String(err));
    }
    const now = portal.now();
    const actor =
      session?.userID ?? (email === "" ? `ip:${context.clientAddress}` : email);
    await checkFeedback(portal.store, { actor, now });
    const organization =
      session == null
        ? null
        : await ensurePersonal(portal.store, {
            userID: session.userID,
            name: session.name,
          });
    const thread = await openThread(portal.store, portal.tracker, {
      kind: "feedback",
      organization,
      title: `Feedback on ${page}`,
      text: description,
      author: session?.name ?? (name || email || ANONYMOUS),
      contact: session?.email ?? email,
      createdBy: session?.userID ?? "",
      page,
      site: portal.site,
      now,
    });
    return Response.json({ thread: thread.key, issue: thread.issueIdentifier });
  } catch (err) {
    return toResponse(err);
  }
};
