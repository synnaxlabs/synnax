# Portal

The portal at `portal.synnaxlabs.com` issues licenses. It deploys like the docs site:
Vercel's Git integration deploys `main` to production and every other branch to a
preview. The Vercel build runs `db:migrate` before it builds, so each migration must
only add to the schema: the old deployment serves on the new schema until the new one is
promoted.

## Development

```sh
pnpm --filter @synnaxlabs/portal dev
```

`.env` holds `DATABASE_URL`, the Clerk development keys, `LICENSE_KMS_KEY_ARN`, and
`STAFF_ORG_ID`. Without `AWS_ROLE_ARN`, the portal signs as your AWS CLI identity.

## Setup

These steps run once, on the Vercel project `portal` with root directory `site/portal`
and domain `portal.synnaxlabs.com`.

### Signing

The production KMS key `alias/synnax-license-signing` signs every license key a Core
accepts. The portal signs through an AWS role that only production deployments can
assume, with Vercel's OIDC tokens, so no AWS secret is stored and a preview cannot sign.
Enable OIDC federation in team issuer mode on the Vercel project, then run:

```sh
VERCEL_TEAM=synnaxlabs site/portal/scripts/create_signing_role.sh
```

`VERCEL_TEAM` is the team slug from the Vercel URL. Set the three values it prints on
every environment of the project.

### Neon

Install Neon from the Vercel Marketplace on the project, one database named `hub`, with
a branch per preview. The install sets `DATABASE_URL`, and the first build migrates it.
Then, with the production URL in the shell:

```sh
STAFF_ORG_ID=org_xxx pnpm --filter @synnaxlabs/portal create-internal-organization
```

It creates the Synnax Labs organization every internal license belongs to.

### Clerk

Install Clerk from the Vercel Marketplace on the project. The install sets
`PUBLIC_CLERK_PUBLISHABLE_KEY` and `CLERK_SECRET_KEY`. In the Clerk dashboard:

1. Organizations: enable them. Create the team organization "Synnax Labs" and give every
   staff member the admin role. Its id is `STAFF_ORG_ID`.
2. Paths: login `/login`, sign-up `/sign-up`, after login `/`.
3. Webhooks: add an endpoint at `https://portal.synnaxlabs.com/api/webhooks/clerk`
   subscribed to `user.created`, `organization.created`, and `organization.updated`. Its
   signing secret is `CLERK_WEBHOOK_SIGNING_SECRET`.
4. Domains: for production, set the application domain to `portal.synnaxlabs.com`, add
   `clerk.portal.synnaxlabs.com`, and add the DNS records it asks for. The CSP in
   `src/middleware.ts` already allows that host.

### Resend

Verify the `synnaxlabs.com` sending domain and create an API key with send permission.
Set it as `RESEND_API_KEY`. Expiry warnings go out from `MAIL_FROM`.

### Cron

Set `CRON_SECRET` to a random string. Vercel sends it to the daily expiry sweep.

### CI license

Log in to the portal as staff, open the Synnax Labs organization's licenses, issue a
subscription of a few months with one node and no channel cap labelled "CI", and use
"Floating license key" on it. Store it with
`gh secret set SYNNAX_LICENSE_TOKEN < <file>`. Rotate it by issuing a new one before the
old one expires.
