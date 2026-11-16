// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { APIRoute } from "astro";

import { licenseFor, requireStaff } from "@/access";
import { handle } from "@/respond";
import { HTTPError } from "@/server/errors";
import { revoke } from "@/server/license/issue";
import { revoked } from "@/server/license/notice";
import { emails } from "@/server/session";

/**
 * POST revokes a license and mails the organization. Staff only. When the mail fails
 * it answers 502 and says the license is revoked, so staff do not revoke it again.
 */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const key = context.params.key ?? "";
    const { portal } = context.locals;
    const session = await portal.session();
    requireStaff(session);
    const { organization } = await licenseFor(portal, session, key);
    // A failed lookup must fail before the change is saved, so a retry is safe.
    const to = await emails(portal.directory, organization);
    const license = await revoke(portal.store, {
      licenseKey: key,
      actor: session.userID,
      now: portal.now(),
    });
    if (to.length > 0)
      try {
        await portal.mail.send({ to, ...revoked(license, organization) });
      } catch (err) {
        throw new HTTPError(
          502,
          `"${license.label}" is revoked, but the email about it failed.`,
          undefined,
          { cause: err },
        );
      }
    return new Response(null, { status: 204 });
  });
