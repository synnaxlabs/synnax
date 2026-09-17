// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { KMSClient } from "@aws-sdk/client-kms";
import { type APIContext } from "astro";
import {
  DATABASE_URL,
  LICENSE_KID,
  LICENSE_KMS_KEY_ARN,
  LINEAR_API_KEY,
  LINEAR_SUPPORT_TEAM,
  MAIL_FROM,
  RESEND_API_KEY,
  STAFF_ORG_ID,
} from "astro:env/server";

import { open as openStore, type Store } from "@/server/db/db";
import { kms, type Signer } from "@/server/license/sign";
import { type Mailer, resend } from "@/server/mail";
import { resolve, type Session } from "@/server/session";
import { linear, type Tracker } from "@/server/support/tracker";

/** Portal is the set of services a portal page or route works with. */
export interface Portal {
  store: Store;
  signer: Signer;
  mail: Mailer;
  tracker: Tracker;
  staffOrgID: string;
  /** site is the origin links in mail and issues are built on. */
  site: string;
  /** session resolves the signed-in user, throwing a 401 when there is none. */
  session: () => Promise<Session>;
  now: () => Date;
}

/**
 * open wires the portal for one request from the runtime environment. Nothing
 * connects until it is used, so docs requests never pay for it.
 */
export const open = (context: APIContext): Portal => ({
  store: openStore(DATABASE_URL),
  signer: kms({
    client: new KMSClient({}),
    keyID: LICENSE_KMS_KEY_ARN,
    kid: LICENSE_KID,
  }),
  mail: resend(RESEND_API_KEY, MAIL_FROM),
  tracker: linear(LINEAR_API_KEY, LINEAR_SUPPORT_TEAM),
  staffOrgID: STAFF_ORG_ID,
  site: context.url.origin,
  session: async () => await resolve(context, STAFF_ORG_ID),
  now: () => new Date(),
});
