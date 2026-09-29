# Infrastructure

One Terraform root per lifecycle. `portal/` owns what the portal signs with and runs on.
State lives in HCP Terraform, organization `synnaxlabs`, workspace `hub`.

## Portal

Terraform declares the AWS signing keys and their IAM identities, the Vercel project
with its environment and domain, and the GitHub Actions secret. Production signs with
the key whose public half the Core embeds. Previews sign with a second key that no Core
trusts. Three vendors stay outside it: Neon and Clerk are installed through the Vercel
Marketplace, which injects their connection string and keys into the project and has no
Terraform surface; Resend has no provider. Their one-time steps are below.

### Apply

```sh
cd infra/portal
terraform init
terraform apply
```

Variables come from HCP Terraform workspace variables: `vercel_team_id`, `staff_org_id`,
`clerk_webhook_signing_secret`, `resend_api_key`, and `ci_license_token`. Provider
credentials come from the environment: `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`
for an administrator, `VERCEL_API_TOKEN`, and `GITHUB_TOKEN`.

Resources that existed before this root did are imported once, never recreated:

```sh
terraform import 'aws_kms_key.license_signing["production"]' <key id>
terraform import 'aws_kms_alias.license_signing["production"]' alias/synnax-license-signing
terraform import github_actions_secret.license_token synnax/SYNNAX_LICENSE_TOKEN
```

A new key spec forces Terraform to replace the key. To move a state that holds an older
key to one created outside Terraform, swap it instead:

```sh
terraform state rm 'aws_kms_key.license_signing["production"]'
terraform import 'aws_kms_key.license_signing["production"]' <new key id>
```

If a `portal` Vercel project already exists, import it the same way:
`terraform import vercel_project.portal <project id>`.

### Neon

Install Neon from the Vercel Marketplace on the portal project, one database named
`hub`. The install sets `DATABASE_URL`. Then, with that URL in the shell:

```sh
pnpm --filter @synnaxlabs/portal db:migrate
STAFF_ORG_ID=org_xxx pnpm --filter @synnaxlabs/portal create-internal-organization
```

`db:migrate` applies the SQL under `site/portal/drizzle/`.
`create-internal-organization` creates the Synnax Labs organization every internal
license belongs to. Rerun `db:migrate` after each schema change lands, and run
`db:generate` to produce the SQL for one.

### Clerk

Install Clerk from the Vercel Marketplace on the portal project. The install sets
`PUBLIC_CLERK_PUBLISHABLE_KEY` and `CLERK_SECRET_KEY` for production and preview. In the
Clerk dashboard:

1. Organizations: enable them. Create the team organization "Synnax Labs" and give every
   staff member the admin role. Its id is `staff_org_id`, the `org_xxx` value above.
2. Paths: login `/login`, sign-up `/sign-up`, after login `/`.
3. Webhooks: add an endpoint at `https://portal.synnaxlabs.com/api/webhooks/clerk`
   subscribed to `user.created`, `organization.created`, and `organization.updated`. Its
   signing secret is `clerk_webhook_signing_secret`.
4. Domains: for production, set the application domain to `portal.synnaxlabs.com`, add
   `clerk.portal.synnaxlabs.com`, and add the DNS records it asks for. The CSP in
   `site/portal/src/middleware.ts` already allows that host.

### Resend

Create the Resend account, verify the `synnaxlabs.com` sending domain, and create an API
key with send permission. The key is `resend_api_key`. Expiry warnings go out from the
address in `mail_from`.

### CI license

Log in to the portal as staff, open the Synnax Labs organization's licenses, issue a
subscription of a few months with one node and no channel cap labelled "CI", and use
"Floating license key" on it. The file's content is `ci_license_token`. Rotate it by
issuing a new one before the old one expires and applying again.
