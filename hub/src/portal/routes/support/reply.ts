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
import { form, handle } from "@/portal/respond";
import { threadFor } from "@/portal/support";
import { badRequest } from "@/server/errors";
import { checkSupport } from "@/server/ratelimit";
import { reply } from "@/server/support/thread";

/** POST appends `message` to a thread as the signed-in viewer. */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const key = context.params.key ?? "";
    const portal = open(context);
    const session = await portal.session();
    const body = await form(context);
    const text = (body.message ?? "").trim();
    if (text === "") throw badRequest("Write a message");
    const view = await threadFor(portal, session, key);
    const now = portal.now();
    await checkSupport(portal.store, { actor: session.userID, now });
    await reply(portal.store, portal.tracker, portal.mail, {
      thread: view.thread,
      status: view.status,
      sender: view.sender,
      author: session.name,
      actor: session.userID,
      text,
      site: portal.site,
      now,
    });
    return new Response(null, { status: 204 });
  });
