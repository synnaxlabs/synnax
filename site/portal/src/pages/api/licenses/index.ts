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
import { badRequest, HTTPError } from "@/server/errors";
import { issue, readTerms } from "@/server/license/issue";
import { issued } from "@/server/license/notice";
import { adoptTeam } from "@/server/organization";
import { emails } from "@/server/session";

/**
 * POST issues a license to the Clerk organization in `organization`, mails the team's
 * admins, and answers `{ key }`. Staff only. When the mail fails it answers 502 and
 * says the license is issued, so staff do not issue it again.
 */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const { portal } = context.locals;
    const session = await portal.session();
    requireStaff(session);
    const body = await form(context);
    if (!body.organization) throw badRequest("Choose an organization");
    const team = await adoptTeam(portal.store, portal.directory, body.organization);
    const license = await issue(portal.store, {
      organization: team.key,
      edition: "enterprise",
      ...readTerms(body),
      actor: session.userID,
      now: portal.now(),
    });
    const to = await emails(portal.directory, team);
    if (to.length > 0)
      try {
        await portal.mail.send({ to, ...issued(license, team) });
      } catch (err) {
        throw new HTTPError(
          502,
          `"${license.label}" is issued, but the email to its admins failed. ` +
            "Open it from the Licenses tab and send them its link.",
          undefined,
          { cause: err },
        );
      }
    return Response.json({ key: license.key });
  });
