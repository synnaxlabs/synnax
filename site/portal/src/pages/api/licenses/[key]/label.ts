// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { isAdmin, licenseFor } from "@/access";
import { form, handle } from "@/respond";
import { forbidden } from "@/server/errors";
import { readLabel, rename } from "@/server/license/issue";

/**
 * POST changes what a license is called and answers with the event it recorded and the
 * name of its actor. Team admins and staff only.
 */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const key = context.params.key ?? "";
    const { portal } = context.locals;
    const session = await portal.session();
    const { organization } = await licenseFor(portal, session, key);
    if (!isAdmin(organization, session))
      throw forbidden("Only a team admin can rename a license");
    const body = await form(context);
    const event = await rename(portal.store, {
      licenseKey: key,
      label: readLabel(body.label),
      actor: session.userID,
      now: portal.now(),
    });
    return Response.json({ event: event ?? null, actor: session.name });
  });
