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
  POSTHOG_API_KEY,
  RESEND_API_KEY,
  SLACK_WEBHOOK_URL,
  STAFF_ORG_ID,
  VERCEL_ENV,
} from "astro:env/server";

import { type Analytics, posthog, unconfigured } from "@/server/analytics";
import { open as openStore, type Store } from "@/server/db/db";
import { clerk, type Directory } from "@/server/directory";
import { kms, type Signer } from "@/server/license/sign";
import { log, type Mailer, resend } from "@/server/mail";
import { resolve, type Session } from "@/server/session";
import { log as logNotes, type Slack, webhook } from "@/server/slack";

/** Portal is the set of services a portal page or route works with. */
export interface Portal {
  store: Store;
  signer: Signer;
  mail: Mailer;
  /** slack posts notable events to the staff channel. */
  slack: Slack;
  /** analytics reads Synnax Desktop sessions from PostHog and records events there. */
  analytics: Analytics;
  directory: Directory;
  staffOrgID: string;
  /** cronSecret is the bearer token Vercel Cron calls the sweep with. */
  cronSecret: string;
  webhookSecret: string;
  /** session resolves the logged-in user, throwing a 401 when there is none. */
  session: () => Promise<Session>;
  /** viewer resolves the logged-in user, or null for a visitor who is not logged in. */
  viewer: () => Promise<Session | null>;
  now: () => Date;
}

/** mailer sends real mail only from production. Previews and local development log it. */
const mailer = (): Mailer => {
  if (VERCEL_ENV !== "production") return log();
  if (RESEND_API_KEY == null)
    throw new Error("RESEND_API_KEY is required in production");
  return resend({
    apiKey: RESEND_API_KEY,
    from: MAIL_FROM,
    origin: import.meta.env.SITE,
  });
};

/**
 * slack posts only from production, and only once its webhook is set. Previews and
 * local development log each note.
 */
const slack = (): Slack =>
  VERCEL_ENV === "production" && SLACK_WEBHOOK_URL != null
    ? webhook({ url: SLACK_WEBHOOK_URL, origin: import.meta.env.SITE })
    : logNotes();

/**
 * open wires the portal for one request from the runtime environment. Nothing
 * connects until it is used, so a request that needs no service pays nothing.
 */
export const open = (context: APIContext): Portal => {
  const directory = clerk(clerkClient(context));
  const session = async (): Promise<Session> =>
    await resolve(directory, context.locals.auth().userId, STAFF_ORG_ID);
  return {
    store: openStore(DATABASE_URL),
    signer: kms({
      // A deployment assumes the signing role through Vercel OIDC. Local development
      // signs as the AWS CLI identity instead.
      client: new KMSClient({
        // Vercel sets AWS_REGION to the function's region, so take the key's own.
        region: LICENSE_KMS_KEY_ARN.split(":")[3],
        credentials:
          AWS_ROLE_ARN == null
            ? undefined
            : awsCredentialsProvider({ roleArn: AWS_ROLE_ARN }),
      }),
      keyID: LICENSE_KMS_KEY_ARN,
      kid: LICENSE_KID,
    }),
    mail: mailer(),
    slack: slack(),
    analytics:
      POSTHOG_API_KEY == null
        ? unconfigured()
        : posthog({
            apiKey: POSTHOG_API_KEY,
            environment: VERCEL_ENV === "production" ? "production" : "development",
          }),
    staffOrgID: STAFF_ORG_ID,
    cronSecret: CRON_SECRET,
    webhookSecret: CLERK_WEBHOOK_SIGNING_SECRET,
    directory,
    session,
    viewer: async () => (context.locals.auth().userId == null ? null : await session()),
    now: () => new Date(),
  };
};
