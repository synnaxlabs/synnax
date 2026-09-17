// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { open } from "@/portal/portal";
import { form, handle, redirect, wantsHTML } from "@/portal/respond";
import { supportFor } from "@/portal/support";
import { badRequest } from "@/server/errors";
import { checkSupport } from "@/server/ratelimit";
import { open as openThread } from "@/server/support/thread";

/** POST opens a thread for the organization in `org` with `title` and `message`. */
export const POST: APIRoute = async (context) =>
  await handle(context, "/support", async () => {
    const portal = open(context);
    const session = await portal.session();
    const body = await form(context);
    const title = (body.title ?? "").trim();
    const text = (body.message ?? "").trim();
    if (title === "") throw badRequest("Give the thread a title");
    if (text === "") throw badRequest("Write a message");
    const { organization } = await supportFor(portal, session, body.org ?? null);
    const now = portal.now();
    await checkSupport(portal.store, { actor: session.userID, now });
    const thread = await openThread(portal.store, portal.tracker, {
      kind: "support",
      organization,
      title,
      text,
      author: session.name,
      contact: session.email,
      createdBy: session.userID,
      site: portal.site,
      now,
    });
    if (wantsHTML(context)) return redirect(context, `/support/${thread.key}`);
    return Response.json({ thread: thread.key, issue: thread.issueIdentifier });
  });
