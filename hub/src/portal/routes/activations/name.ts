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
import { form, handle } from "@/portal/respond";
import { readName, rename } from "@/server/license/machine";

/** POST changes what a machine is called. */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const key = context.params.key ?? "";
    const portal = open(context);
    const session = await portal.session();
    await activationFor(portal, session, key);
    const body = await form(context);
    await rename(portal.store, {
      activationKey: key,
      name: readName(body.name),
      actor: session.userID,
    });
    return new Response(null, { status: 204 });
  });
