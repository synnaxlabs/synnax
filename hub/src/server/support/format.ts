// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type License, type Organization, type Sender } from "@/server/db/schema";

export interface OpenArgs {
  author: string;
  contact: string;
  text: string;
  site: string;
  organization: Organization | null;
  licenses: License[];
  /** page is the docs path a feedback submission came from. */
  page?: string;
  now: Date;
}

const date = (d: Date | null): string =>
  d == null ? "" : d.toISOString().slice(0, 10);

const licenseLine = (lic: License, now: Date): string => {
  const state =
    lic.revokedAt != null
      ? "revoked"
      : lic.expiresAt != null && lic.expiresAt <= now
        ? "expired"
        : "active";
  const term =
    lic.term === "perpetual"
      ? `perpetual up to ${lic.maxVersion}`
      : `until ${date(lic.expiresAt)}`;
  const name = lic.label || lic.key;
  return `- ${name}: ${lic.edition}, ${lic.nodes} nodes, ${term}, ${state}`;
};

/**
 * description writes the issue body staff read first: who wrote in, their
 * organization with a portal link, their licenses, and the message.
 */
export const description = ({
  author,
  contact,
  text,
  site,
  organization,
  licenses,
  page,
  now,
}: OpenArgs): string => {
  const who = contact === "" ? author : `${author} (${contact})`;
  const lines = [`Opened by ${who} from the portal.`];
  if (page != null) lines.push(`Page: ${site}${page}`);
  if (organization != null)
    lines.push(
      `Organization: [${organization.name}](${site}/licenses?org=${organization.key})`,
    );
  if (licenses.length > 0)
    lines.push("", "Licenses:", ...licenses.map((lic) => licenseLine(lic, now)));
  return [...lines, "", "---", "", text].join("\n");
};

/** comment writes a message as staff see it on the issue. */
export const comment = (sender: Sender, author: string, text: string): string =>
  `**${author}** (${sender}, via the portal)\n\n${text}`;

export interface ReplyMailArgs {
  title: string;
  author: string;
  text: string;
  url: string;
}

/** replyMail is the notice a member gets when staff answer their thread. */
export const replyMail = ({
  title,
  author,
  text,
  url,
}: ReplyMailArgs): { subject: string; text: string } => ({
  subject: `Re: ${title}`,
  text: `${author} from Synnax replied to "${title}":\n\n${text}\n\nReply at ${url}`,
});
