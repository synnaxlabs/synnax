// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { auth } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import type { ReactElement } from "react";

interface Brand {
  name: string;
  icon: ReactElement;
  // provider names that mean this brand, in the form normalize returns
  aliases: string[];
  // a leading dot matches any subdomain
  hosts: string[];
}

const BRANDS: Brand[] = [
  {
    name: "Amazon Cognito",
    icon: <Icon.Logo.AWS />,
    aliases: ["cognito", "amazoncognito", "aws", "awscognito"],
    hosts: [".amazoncognito.com"],
  },
  {
    name: "Apple",
    icon: <Icon.Logo.Apple />,
    aliases: ["apple"],
    hosts: ["appleid.apple.com"],
  },
  {
    name: "Auth0",
    icon: <Icon.Logo.Auth0 />,
    aliases: ["auth0"],
    hosts: [".auth0.com"],
  },
  { name: "Authelia", icon: <Icon.Logo.Authelia />, aliases: ["authelia"], hosts: [] },
  {
    name: "Authentik",
    icon: <Icon.Logo.Authentik />,
    aliases: ["authentik"],
    hosts: [],
  },
  {
    name: "Cloudflare",
    icon: <Icon.Logo.Cloudflare />,
    aliases: ["cloudflare", "cloudflareaccess"],
    hosts: [".cloudflareaccess.com"],
  },
  {
    name: "Duo",
    icon: <Icon.Logo.Duo />,
    aliases: ["duo", "ciscoduo"],
    hosts: [".duosecurity.com"],
  },
  {
    name: "FusionAuth",
    icon: <Icon.Logo.FusionAuth />,
    aliases: ["fusionauth"],
    hosts: [".fusionauth.io"],
  },
  { name: "Gitea", icon: <Icon.Logo.Gitea />, aliases: ["gitea"], hosts: [] },
  {
    name: "GitHub",
    icon: <Icon.Logo.Github />,
    aliases: ["github"],
    hosts: ["github.com"],
  },
  {
    name: "GitLab",
    icon: <Icon.Logo.GitLab />,
    aliases: ["gitlab"],
    hosts: ["gitlab.com"],
  },
  {
    name: "Google",
    icon: <Icon.Logo.Google />,
    aliases: ["google", "googleworkspace"],
    hosts: ["accounts.google.com"],
  },
  {
    name: "JumpCloud",
    icon: <Icon.Logo.JumpCloud />,
    aliases: ["jumpcloud"],
    hosts: ["oauth.id.jumpcloud.com"],
  },
  { name: "Keycloak", icon: <Icon.Logo.Keycloak />, aliases: ["keycloak"], hosts: [] },
  {
    name: "Microsoft",
    icon: <Icon.Logo.Microsoft />,
    aliases: ["microsoft", "entra", "entraid", "microsoftentraid", "azure", "azuread"],
    hosts: ["login.microsoftonline.com"],
  },
  {
    name: "Okta",
    icon: <Icon.Logo.Okta />,
    aliases: ["okta"],
    hosts: [".okta.com", ".oktapreview.com", ".okta-emea.com"],
  },
  {
    name: "OneLogin",
    icon: <Icon.Logo.OneLogin />,
    aliases: ["onelogin"],
    hosts: [".onelogin.com"],
  },
  { name: "Ory", icon: <Icon.Logo.Ory />, aliases: ["ory"], hosts: [".oryapis.com"] },
  {
    name: "Ping Identity",
    icon: <Icon.Logo.PingIdentity />,
    aliases: ["ping", "pingidentity", "pingone", "pingfederate"],
    hosts: [".pingone.com", ".pingone.eu", ".pingone.ca", ".pingone.asia"],
  },
  {
    name: "Salesforce",
    icon: <Icon.Logo.Salesforce />,
    aliases: ["salesforce"],
    hosts: [".salesforce.com", ".force.com"],
  },
  {
    name: "Zitadel",
    icon: <Icon.Logo.Zitadel />,
    aliases: ["zitadel"],
    hosts: [".zitadel.cloud"],
  },
];

const FALLBACK_ICON = <Icon.Logo.OpenID />;

// "Azure AD", "azure-ad" and "azure_ad" all become "azuread"
const normalize = (name: string): string =>
  name.toLowerCase().replace(/[^a-z0-9]/g, "");

const hostMatches = (host: string, pattern: string): boolean =>
  pattern.startsWith(".") ? host.endsWith(pattern) : host === pattern;

const brandOfHost = (endpoint: string): Brand | undefined => {
  let host: string;
  try {
    host = new URL(endpoint).hostname;
  } catch {
    return undefined;
  }
  return BRANDS.find((b) => b.hosts.some((p) => hostMatches(host, p)));
};

/** How the login page shows a provider: its label and its logo. */
export interface Display {
  label: string;
  icon: ReactElement;
}

/**
 * Shows a provider named after a known brand, such as "google" or "azure-ad", with the
 * brand's name and logo. A provider on a known brand's login host keeps its own name
 * and gets the logo, so two providers on the same brand stay apart. Any other provider
 * keeps its name and gets the OpenID logo.
 */
export const display = (login: auth.OIDCLogin): Display => {
  const key = normalize(login.provider);
  const named = BRANDS.find((b) => b.aliases.includes(key));
  if (named != null) return { label: named.name, icon: named.icon };
  const icon = brandOfHost(login.authorizationEndpoint)?.icon ?? FALLBACK_ICON;
  return { label: login.provider, icon };
};
