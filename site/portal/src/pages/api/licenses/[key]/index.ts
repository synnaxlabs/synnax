// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { licenseFor, requireStaff } from "@/access";
import { form, handle } from "@/respond";
import { amend, readTerms } from "@/server/license/issue";

/** POST changes the terms of a license already issued. Staff only. */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const key = context.params.key ?? "";
    const { portal } = context.locals;
    const session = await portal.session();
    requireStaff(session);
    await licenseFor(portal, session, key);
    const body = await form(context);
    await amend(portal.store, {
      licenseKey: key,
      ...readTerms(body),
      actor: session.userID,
      now: portal.now(),
    });
    return new Response(null, { status: 204 });
  });
