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
import { download, handle } from "@/portal/respond";
import { floating } from "@/server/license/issue";

/** POST downloads a token bound to no machine, for CI runners. Staff only. */
export const POST: APIRoute = async (context) => {
  const key = context.params.key ?? "";
  return await handle(context, `/licenses/${key}`, async () => {
    const portal = open(context);
    const session = await portal.session();
    requireStaff(session);
    const { license } = await licenseFor(portal, session, key);
    const token = await floating(portal.store, portal.signer, {
      licenseKey: key,
      actor: session.userID,
      now: portal.now(),
    });
    return download(token, license.label);
  });
};
