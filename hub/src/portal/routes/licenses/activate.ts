// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { licenseFor } from "@/portal/access";
import { open } from "@/portal/portal";
import { filename, form, handle } from "@/portal/respond";
import { badRequest } from "@/server/errors";
import { activate, DENIAL_MESSAGES } from "@/server/license/activate";
import { parse } from "@/server/license/fingerprint";
import { check } from "@/server/ratelimit";

/**
 * POST grants a seat to the machine whose host hashes are posted as `fingerprint` and
 * answers `{ token, activation, filename }`.
 */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const key = context.params.key ?? "";
    const portal = open(context);
    const session = await portal.session();
    const { license, organization } = await licenseFor(portal, session, key);
    const body = await form(context);
    let fingerprint: string[];
    try {
      fingerprint = parse(body.fingerprint ?? "");
    } catch (err) {
      throw badRequest((err as Error).message);
    }
    const now = portal.now();
    await check(portal.store, {
      actor: session.userID,
      organization: organization.key,
      now,
    });
    const result = await activate(portal.store, portal.signer, {
      licenseKey: license.key,
      fingerprint,
      actor: session.userID,
      now,
    });
    if (!result.ok) throw badRequest(DENIAL_MESSAGES[result.reason]);
    return Response.json({
      token: result.token,
      activation: result.activation.key,
      filename: filename(license.label),
    });
  });
