# Portal

The portal at `portal.synnaxlabs.com` issues licenses. Vercel's Git integration deploys
`main` to production and every other branch to a preview. The Vercel build migrates the
database after it builds, so each migration must only add to the schema: the old
deployment serves on the new schema until the new one is promoted.

## Environments

| Environment | Database              | Clerk                | Signs keys | Sends mail |
| ----------- | --------------------- | -------------------- | ---------- | ---------- |
| Production  | Neon `hub`, `main`    | Production instance  | Yes        | Yes        |
| Preview     | Neon `hub`, `preview` | Development instance | No         | No, logs   |
| Development | Neon `hub`, `preview` | Development instance | Yes        | No, logs   |

Every preview shares the `preview` branch, so an unmerged migration never reaches
production. A preview cannot sign, because only production deployments can assume the
signing role: routes that return a license key fail there. Previews are visible only to
members of the Vercel team.

## Development

```sh
pnpm --filter @synnaxlabs/portal dev
```

`.env` in this folder holds:

- `DATABASE_URL`: the `preview` branch of the Neon `hub` project.
- `PUBLIC_CLERK_PUBLISHABLE_KEY`, `CLERK_SECRET_KEY`: the Clerk development instance.
- `STAFF_ORG_ID`: the "Synnax Labs" organization in the development instance.
- `LICENSE_KMS_KEY_ARN`: the signing key. The portal signs as your AWS CLI identity,
  which needs `kms:Sign` on it.
- `CLERK_WEBHOOK_SIGNING_SECRET`, `CRON_SECRET`: any random strings.

## Setup

These steps run once. They create the Vercel project `portal` and the services it uses.

### Signing

The KMS key `alias/synnax-license-signing` signs every license key a Core accepts. The
portal signs through an AWS role that only production deployments of the project can
assume with Vercel's OIDC tokens, so no AWS secret is stored. With an AWS CLI identity
that can manage IAM, run:

```sh
site/portal/scripts/create_signing_role.sh
```

It prints `AWS_ROLE_ARN` and `LICENSE_KMS_KEY_ARN` for the production environment.

### Neon

The Neon project `hub` comes from the Vercel Marketplace. Connect it to the `portal`
project for production only, which sets `DATABASE_URL` there. Create one branch named
`preview` from `main`, and set its connection string as `DATABASE_URL` on the preview
and development environments.

Before the first production deploy, with the production URL in the shell, run:

```sh
pnpm --filter @synnaxlabs/portal db:migrate
STAFF_ORG_ID=org_xxx pnpm --filter @synnaxlabs/portal create-internal-organization
```

It creates the Synnax Labs organization every internal license belongs to. It must run
before anyone opens the portal, which otherwise adopts the organization under a new key.

### Clerk

The Clerk application `hub` has a production instance and a development instance. In
each instance:

1. Organizations: enable them. Create the team organization "Synnax Labs" and give every
   staff member the admin role. Its id is that environment's `STAFF_ORG_ID`.
2. Paths: sign-in `/sign-in`, sign-up `/sign-up`, after sign-in `/`.
3. Sign-in options: Google and Microsoft. Production needs its own OAuth credentials for
   both, with the redirect URLs the dashboard shows.

In the production instance only:

1. Domains: set the application domain to `synnaxlabs.com`, and add the DNS records it
   asks for. `src/middleware.ts` allows its Frontend API host, `clerk.synnaxlabs.com`.
2. Webhooks: add an endpoint at `https://portal.synnaxlabs.com/api/webhooks/clerk`
   subscribed to `user.created`, `organization.created`, and `organization.updated`. Its
   signing secret is `CLERK_WEBHOOK_SIGNING_SECRET`.

Previews receive no webhooks. The portal creates organizations on first use instead.

### Resend

Verify the `synnaxlabs.com` sending domain and create an API key with send permission
for it. Set it as `RESEND_API_KEY` on production only. Expiry and revocation notices go
out from `MAIL_FROM`.

### Vercel

Create the project `portal` from the `synnaxlabs/synnax` repository with root directory
`site/portal`, production branch `main`, files outside the root directory included, and
OIDC federation in team issuer mode. `vercel.json` sets the install, build, and cron.

| Variable                       | Production      | Preview and development |
| ------------------------------ | --------------- | ----------------------- |
| `DATABASE_URL`                 | Neon `main`     | Neon `preview`          |
| `PUBLIC_CLERK_PUBLISHABLE_KEY` | Production key  | Development key         |
| `CLERK_SECRET_KEY`             | Production key  | Development key         |
| `STAFF_ORG_ID`                 | Production org  | Development org         |
| `CLERK_WEBHOOK_SIGNING_SECRET` | Webhook secret  | Random string           |
| `LICENSE_KMS_KEY_ARN`          | From the script | From the script         |
| `AWS_ROLE_ARN`                 | From the script | Unset                   |
| `RESEND_API_KEY`               | Resend key      | Unset                   |
| `CRON_SECRET`                  | Random string   | Random string           |

Add the domain `portal.synnaxlabs.com`, and point a `portal` CNAME at the value Vercel
shows. Vercel Cron calls the daily expiry sweep on production with `CRON_SECRET`.

### CI license

Sign in to the portal as staff, open the Synnax Labs organization's licenses, issue a
subscription of a few months with one node and no channel cap labelled "CI", and use
"Floating license key" on it. Store it with
`gh secret set SYNNAX_LICENSE_TOKEN < <file>`. Rotate it by issuing a new one before the
old one expires.
