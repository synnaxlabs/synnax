// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { requireStaff } from "@/access";
import { form, handle } from "@/respond";
import { badRequest } from "@/server/errors";
import { issue, readTerms } from "@/server/license/issue";
import { adoptTeam } from "@/server/organization";

/**
 * POST issues a license to the Clerk organization in `organization` and answers
 * `{ key }`. Staff only.
 */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const { portal } = context.locals;
    const session = await portal.session();
    requireStaff(session);
    const body = await form(context);
    if (!body.organization) throw badRequest("Choose an organization");
    const team = await adoptTeam(portal.store, portal.directory, body.organization);
    const { key } = await issue(portal.store, {
      organization: team.key,
      edition: "enterprise",
      ...readTerms(body),
      actor: session.userID,
      now: portal.now(),
    });
    return Response.json({ key });
  });
