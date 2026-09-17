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
import { threadFor } from "@/portal/support";
import { badRequest, forbidden } from "@/server/errors";
import { checkSupport } from "@/server/ratelimit";
import { record } from "@/server/support/record";

/** POST appends `message` to a thread as the signed-in member. */
export const POST: APIRoute = async (context) => {
  const id = context.params.id ?? "";
  return await handle(context, `/support/${id}`, async () => {
    const portal = open(context);
    const session = await portal.session();
    const body = await form(context);
    const text = (body.message ?? "").trim();
    if (text === "") throw badRequest("Write a message");
    const { thread, organization, customerID } = await threadFor(portal, session, id);
    if (customerID == null) throw forbidden("Staff reply from Plain");
    await checkSupport(portal.store, { actor: session.userID, now: portal.now() });
    await portal.support.reply({ customerID, threadID: thread.id, text });
    await record(portal.store, {
      kind: "thread",
      actor: session.userID,
      organization: organization.key,
      detail: { thread: thread.id, reply: true },
    });
    if (wantsHTML(context)) return redirect(context, `/support/${thread.id}`);
    return new Response(null, { status: 204 });
  });
};
