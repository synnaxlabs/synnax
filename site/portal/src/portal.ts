// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { KMSClient } from "@aws-sdk/client-kms";
import { clerkClient } from "@clerk/astro/server";
import { awsCredentialsProvider } from "@vercel/oidc-aws-credentials-provider";
import { type APIContext } from "astro";
import {
  AWS_ROLE_ARN,
  CLERK_WEBHOOK_SIGNING_SECRET,
  CRON_SECRET,
  DATABASE_URL,
  LICENSE_KID,
  LICENSE_KMS_KEY_ARN,
  MAIL_FROM,
  RESEND_API_KEY,
  STAFF_ORG_ID,
} from "astro:env/server";

import { open as openStore, type Store } from "@/server/db/db";
import { clerk, type Directory } from "@/server/directory";
import { kms, type Signer } from "@/server/license/sign";
import { type Mailer, resend } from "@/server/mail";
import { resolve, type Session } from "@/server/session";

/** Portal is the set of services a portal page or route works with. */
export interface Portal {
  store: Store;
  signer: Signer;
  mail: Mailer;
  directory: Directory;
  staffOrgID: string;
  /** cronSecret is the bearer token Vercel Cron calls the sweep with. */
  cronSecret: string;
  webhookSecret: string;
  /** session resolves the logged-in user, throwing a 401 when there is none. */
  session: () => Promise<Session>;
  now: () => Date;
}

/**
 * open wires the portal for one request from the runtime environment. Nothing
 * connects until it is used, so a request that needs no service pays nothing.
 */
export const open = (context: APIContext): Portal => {
  const directory = clerk(clerkClient(context));
  return {
    store: openStore(DATABASE_URL),
    signer: kms({
      // A deployment assumes the signing role through Vercel OIDC. Local development
      // signs as the AWS CLI identity instead.
      client: new KMSClient({
        credentials:
          AWS_ROLE_ARN == null
            ? undefined
            : awsCredentialsProvider({ roleArn: AWS_ROLE_ARN }),
      }),
      keyID: LICENSE_KMS_KEY_ARN,
      kid: LICENSE_KID,
    }),
    mail: resend(RESEND_API_KEY, MAIL_FROM),
    staffOrgID: STAFF_ORG_ID,
    cronSecret: CRON_SECRET,
    webhookSecret: CLERK_WEBHOOK_SIGNING_SECRET,
    directory,
    session: async () =>
      await resolve(directory, context.locals.auth().userId, STAFF_ORG_ID),
    now: () => new Date(),
  };
};
