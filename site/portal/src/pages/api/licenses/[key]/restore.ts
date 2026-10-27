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
import { handle } from "@/respond";
import { restore } from "@/server/license/issue";
import { restored } from "@/server/license/notice";
import { emails } from "@/server/session";

/** POST restores a revoked license and mails the organization. Staff only. */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const key = context.params.key ?? "";
    const { portal } = context.locals;
    const session = await portal.session();
    requireStaff(session);
    const { organization } = await licenseFor(portal, session, key);
    const license = await restore(portal.store, {
      licenseKey: key,
      actor: session.userID,
      now: portal.now(),
    });
    const to = await emails(portal.directory, organization);
    if (to.length > 0)
      await portal.mail.send({ to, ...restored(license, organization) });
    return new Response(null, { status: 204 });
  });
