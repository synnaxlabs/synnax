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
import { record } from "@/server/support/record";

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
    const { organization, customerID } = await supportFor(
      portal,
      session,
      body.org ?? null,
    );
    const now = portal.now();
    await checkSupport(portal.store, { actor: session.userID, now });
    const thread = await portal.support.createThread({
      customerID,
      organizationKey: organization.key,
      title,
      text,
    });
    await record(portal.store, {
      kind: "thread",
      actor: session.userID,
      organization: organization.key,
      detail: { thread: thread.id },
    });
    if (wantsHTML(context)) return redirect(context, `/support/${thread.id}`);
    return Response.json({ thread: thread.id, ref: thread.ref });
  });
