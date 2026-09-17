// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { activationFor } from "@/portal/access";
import { open } from "@/portal/portal";
import { download, handle } from "@/portal/respond";
import { reissue } from "@/server/license/activate";
import { check } from "@/server/ratelimit";

/** GET downloads a fresh token for a machine that holds a seat. */
export const GET: APIRoute = async (context) =>
  await handle(async () => {
    const key = context.params.key ?? "";
    const portal = open(context);
    const session = await portal.session();
    const { license, organization } = await activationFor(portal, session, key);
    const now = portal.now();
    await check(portal.store, {
      actor: session.userID,
      organization: organization.key,
      now,
    });
    const { token } = await reissue(portal.store, portal.signer, {
      activationKey: key,
      actor: session.userID,
      now,
    });
    return download(token, license.label);
  });
