// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createHmac, timingSafeEqual } from "node:crypto";

import { z } from "zod";

import { type Activation, type License, type Organization } from "@/server/db/schema";

/** SIGNATURE_HEADER carries Plain's HMAC-SHA256 of the request body, hex encoded. */
export const SIGNATURE_HEADER = "plain-request-signature";

/** verify reports whether a request body carries a valid signature under the secret. */
export const verify = (
  secret: string,
  body: string,
  signature: string | null,
): boolean => {
  if (signature == null) return false;
  const expected = createHmac("sha256", secret).update(body).digest("hex");
  const given = Buffer.from(signature, "utf8");
  const want = Buffer.from(expected, "utf8");
  return given.length === want.length && timingSafeEqual(given, want);
};

export const requestZ = z.object({
  cardKeys: z.array(z.string()),
  customer: z.object({
    id: z.string(),
    email: z.string(),
    externalId: z.string().nullable(),
  }),
});
export type Request = z.infer<typeof requestZ>;

export const CARD_KEYS = ["organizations", "licenses", "activations"] as const;
export type CardKey = (typeof CARD_KEYS)[number];

/** TTL_SECONDS is how long Plain caches a card before asking again. */
export const TTL_SECONDS = 300;

export interface Component {
  componentText?: { text: string; textSize?: "S" | "M" | "L"; textColor?: string };
  componentRow?: { rowMainContent: Component[]; rowAsideContent: Component[] };
  componentBadge?: {
    badgeLabel: string;
    badgeColor?: "GREY" | "GREEN" | "YELLOW" | "RED" | "BLUE";
  };
  componentDivider?: { dividerSpacingSize?: "XS" | "S" | "M" | "L" | "XL" };
  componentLinkButton?: { linkButtonUrl: string; linkButtonLabel: string };
}

export interface Card {
  key: string;
  timeToLiveSeconds: number;
  components: Component[];
}

export interface View {
  organizations: Organization[];
  licenses: (License & { seats: number; organizationName: string })[];
  activations: (Activation & { label: string })[];
  /** site is the origin portal links are built on. */
  site: string;
  now: Date;
}

const text = (s: string, textColor?: string): Component => ({
  componentText: { text: s, ...(textColor == null ? {} : { textColor }) },
});
const muted = (s: string): Component => text(s, "MUTED");
const row = (main: Component, aside: Component): Component => ({
  componentRow: { rowMainContent: [main], rowAsideContent: [aside] },
});
const badge = (
  label: string,
  color: NonNullable<Component["componentBadge"]>["badgeColor"],
): Component => ({
  componentBadge: { badgeLabel: label, badgeColor: color },
});
const link = (url: string, label: string): Component => ({
  componentLinkButton: { linkButtonUrl: url, linkButtonLabel: label },
});
const date = (d: Date | null): string =>
  d == null ? "" : d.toISOString().slice(0, 10);

const status = (lic: License, now: Date): Component => {
  if (lic.revokedAt != null) return badge("Revoked", "RED");
  if (lic.expiresAt != null && lic.expiresAt <= now) return badge("Expired", "YELLOW");
  return badge("Active", "GREEN");
};

const organizations = ({ organizations, site }: View): Component[] =>
  organizations.length === 0
    ? [muted("No portal organization")]
    : organizations.flatMap((org) => [
        row(text(`**${org.name}**`), badge(org.kind, "GREY")),
        link(`${site}/licenses?org=${org.key}`, "Open in portal"),
      ]);

const licenses = ({ licenses, site, now }: View): Component[] =>
  licenses.length === 0
    ? [muted("No licenses")]
    : licenses.flatMap((lic, i) => {
        const until = lic.expiresAt == null ? "" : ` · until ${date(lic.expiresAt)}`;
        const seats = `${lic.seats} of ${lic.nodes} seats`;
        return [
          ...(i === 0
            ? []
            : [{ componentDivider: { dividerSpacingSize: "S" as const } }]),
          row(text(`**${lic.label || lic.key}**`), status(lic, now)),
          muted(
            `${lic.organizationName} · ${lic.edition} · ${lic.term} · ${seats}${until}`,
          ),
          link(`${site}/licenses/${lic.key}`, "Open license"),
        ];
      });

const activations = ({ activations }: View): Component[] =>
  activations.length === 0
    ? [muted("No activations")]
    : activations.map((act) => {
        const hash = act.fingerprint[0]?.slice(0, 12) ?? "floating";
        const more =
          act.fingerprint.length > 1 ? ` +${act.fingerprint.length - 1}` : "";
        return row(
          text(`**${act.label}** \`${hash}\`${more}`),
          act.releasedAt == null
            ? muted(`last token ${date(act.lastSeen)}`)
            : badge(`released ${date(act.releasedAt)}`, "GREY"),
        );
      });

const BUILDERS: Record<CardKey, (view: View) => Component[]> = {
  organizations,
  licenses,
  activations,
};

/**
 * build answers every requested key, since Plain marks a card missing from the
 * response as an integration error. A key this site does not serve gets an empty card.
 */
export const build = (keys: string[], view: View): Card[] =>
  keys.map((key) => ({
    key,
    timeToLiveSeconds: TTL_SECONDS,
    components: key in BUILDERS ? BUILDERS[key as CardKey](view) : [],
  }));
