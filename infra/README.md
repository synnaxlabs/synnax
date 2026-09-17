# Infrastructure

One Terraform root per lifecycle. `hub/` owns what the hub portal signs with and runs on.
State lives in HCP Terraform, organization `synnaxlabs`, workspace `hub`.

## Hub

Terraform declares the AWS signing key and its IAM identity, the Vercel environment and
domain, and the GitHub Actions secret. Three vendors stay outside it: Neon and Clerk are
installed through the Vercel Marketplace, which injects their connection string and keys
into the project and has no Terraform surface; Resend and Plain have no provider. Their
one-time steps are below.

### Apply

```sh
cd infra/hub
terraform init
terraform apply
```

Variables come from HCP Terraform workspace variables: `vercel_team_id`, `staff_org_id`,
`clerk_webhook_signing_secret`, `resend_api_key`, and `ci_license_token`. Provider
credentials come from the environment: `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`
for an administrator, `VERCEL_API_TOKEN`, and `GITHUB_TOKEN`.

Resources that existed before this root did are imported once, never recreated:

```sh
terraform import aws_kms_key.license_signing <key id>
terraform import aws_kms_alias.license_signing alias/synnax-license-signing
terraform import 'vercel_project_domain.hub' <project id>/docs.synnaxlabs.com
terraform import github_actions_secret.license_token synnax/SYNNAX_LICENSE_TOKEN
```

### Neon

Install Neon from the Vercel Marketplace on the hub project, one database named `hub`,
preview branching on. The install sets `DATABASE_URL`. Then, with that URL in the shell:

```sh
pnpm --filter @synnaxlabs/hub db:migrate
STAFF_ORG_ID=<clerk organization id> pnpm --filter @synnaxlabs/hub seed
```

`db:migrate` applies the SQL under `hub/drizzle/`. `seed` creates the Synnax Labs
organization every internal license belongs to. Rerun `db:migrate` after each schema
change lands, and run `db:generate` to produce the SQL for one.

### Clerk

Install Clerk from the Vercel Marketplace on the hub project. The install sets
`PUBLIC_CLERK_PUBLISHABLE_KEY` and `CLERK_SECRET_KEY` for production and preview. In the
Clerk dashboard:

1. Organizations: enable them. Create the team organization "Synnax Labs" and give
   every staff member the admin role. Its id is `staff_org_id`.
2. Paths: sign-in `/sign-in`, sign-up `/sign-up`, after sign-in `/account`.
3. Webhooks: add an endpoint at `https://docs.synnaxlabs.com/api/webhooks/clerk`
   subscribed to `user.created`, `organization.created`, and `organization.updated`.
   Its signing secret is `clerk_webhook_signing_secret`.
4. Domains: for production, add `clerk.docs.synnaxlabs.com` and the DNS records it
   asks for. The CSP in `hub/src/middleware.ts` already allows that host.

### Resend

Create the Resend account, verify the `synnaxlabs.com` sending domain, and create an API
key with send permission. The key is `resend_api_key`. Expiry warnings go out from the
address in `mail_from`.

### CI license

Sign in to the portal as staff, open the Synnax Labs organization's licenses, issue a
subscription of a few months with one node and no channel cap labelled "CI", and use
"Download floating token" on it. The file's content is `ci_license_token`. Rotate it by
issuing a new one before the old one expires and applying again.

### Plain

Plain has no Terraform provider. Its setup is documented with the support work.
