// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { licenseFor } from "@/access";
import { form, handle } from "@/respond";
import { isInternal } from "@/server/directory";
import { badRequest } from "@/server/errors";
import { activate } from "@/server/license/activate";
import { DENIAL_MESSAGES } from "@/server/license/deny";
import { parse } from "@/server/license/fingerprint";
import { offer, readCode } from "@/server/license/handoff";
import { filename } from "@/server/license/limits";
import { readName } from "@/server/license/machine";
import { check } from "@/server/ratelimit";
import { refused } from "@/server/slack";

/**
 * POST grants a seat to the machine whose fingerprint is posted as `fingerprint` and
 * answers `{ key, activation, filename }`. With a `handoff` code, it also holds the key
 * for the Console that made the code.
 */
export const POST: APIRoute = async (context) =>
  await handle(async () => {
    const key = context.params.key ?? "";
    const { portal } = context.locals;
    const session = await portal.session();
    const { license, organization } = await licenseFor(portal, session, key);
    const body = await form(context);
    let fingerprint: string[];
    try {
      fingerprint = parse(body.fingerprint ?? "");
    } catch (err) {
      throw badRequest((err as Error).message);
    }
    const name = readName(body.name);
    const code = body.handoff == null ? null : readCode(body.handoff);
    const now = portal.now();
    await check(portal.store, {
      actor: session.userID,
      organization: organization.key,
      now,
    });
    const result = await activate(portal.store, portal.signer, {
      licenseKey: license.key,
      fingerprint,
      name,
      actor: session.userID,
      now,
    });
    if (!result.ok) {
      if (!isInternal(session.email))
        await portal.slack.post(
          refused({ organization, license, reason: result.reason, actor: session }),
        );
      throw badRequest(DENIAL_MESSAGES[result.reason]);
    }
    if (code != null) await offer(portal.store, { code, key: result.key, now });
    return Response.json({
      key: result.key,
      activation: result.activation.key,
      filename: filename(license.label, name),
    });
  });
