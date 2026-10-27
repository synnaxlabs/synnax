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
import { register } from "@/server/organization";

/**
 * POST registers a team named `name`, invites `email` as its first admin, and
 * answers `{ key }`. Staff only.
 */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const { portal } = context.locals;
    const session = await portal.session();
    requireStaff(session);
    const body = await form(context);
    const { key } = await register(portal.store, portal.directory, portal.mail, {
      name: body.name ?? "",
      email: body.email ?? "",
      actor: session.userID,
      now: portal.now(),
    });
    return Response.json({ key });
  });
