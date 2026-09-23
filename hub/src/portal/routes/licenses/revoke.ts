// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { licenseFor, requireStaff } from "@/portal/access";
import { open } from "@/portal/portal";
import { handle } from "@/portal/respond";
import { revocationText } from "@/server/license/expiry";
import { revoke } from "@/server/license/issue";
import { emails } from "@/server/session";

/** POST revokes a license and mails the organization. Staff only. */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const key = context.params.key ?? "";
    const portal = open(context);
    const session = await portal.session();
    requireStaff(session);
    const { organization } = await licenseFor(portal, session, key);
    const license = await revoke(portal.store, {
      licenseKey: key,
      actor: session.userID,
      now: portal.now(),
    });
    const to = await emails(context, organization);
    if (to.length > 0)
      await portal.mail.send({
        to,
        subject: `Your Synnax license "${license.label}" was revoked`,
        text: revocationText(license, organization.name),
      });
    return new Response(null, { status: 204 });
  });
