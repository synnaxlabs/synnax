// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CSSProperties, type ReactElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { Resend } from "resend";

export interface Fact {
  label: string;
  value: string;
}

export interface Action {
  label: string;
  /** href is a path on the portal, such as `/licenses/<key>`. */
  href: string;
}

/** Notice is the content of one email. The mailer lays it out as HTML and as text. */
export interface Notice {
  subject: string;
  heading: string;
  /** paragraphs read under the heading. Inboxes preview the first one. */
  paragraphs: string[];
  facts: Fact[];
  action: Action;
  /** reason tells the reader why they get the email. */
  reason: string;
}

export interface Message extends Notice {
  to: string[];
}

/** Mailer sends license notices. Clerk sends the rest of the portal's mail. */
export interface Mailer {
  send: (message: Message) => Promise<void>;
}

export interface Rendered {
  text: string;
  html: string;
}

const COLOR = {
  // public/email/logo.png has this color baked in, so dark mode clients that darken
  // the page cannot hide the wordmark.
  page: "#f4f5f7",
  card: "#ffffff",
  panel: "#f8f9fb",
  border: "#e4e6ea",
  heading: "#07080a",
  text: "#3b3e44",
  muted: "#63666c",
  primary: "#3774d0",
};

const FONT =
  "Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, " +
  "sans-serif";

const TEXT: CSSProperties = {
  margin: "0 0 16px",
  fontSize: 14,
  lineHeight: "22px",
  color: COLOR.text,
};

// Mail clients drop most CSS, so the layout is tables with inline styles.
const Email = ({
  notice,
  origin,
}: {
  notice: Notice;
  origin: string;
}): ReactElement => (
  <html lang="en">
    <head>
      <meta httpEquiv="Content-Type" content="text/html; charset=UTF-8" />
      <meta name="viewport" content="width=device-width, initial-scale=1" />
      <meta name="color-scheme" content="light" />
      <meta name="supported-color-schemes" content="light" />
      <title>{notice.subject}</title>
    </head>
    <body style={{ margin: 0, padding: 0, background: COLOR.page, fontFamily: FONT }}>
      <div style={{ display: "none", maxHeight: 0, overflow: "hidden" }}>
        {notice.paragraphs[0]}
      </div>
      <table
        role="presentation"
        width="100%"
        cellPadding={0}
        cellSpacing={0}
        style={{ background: COLOR.page }}
      >
        <tbody>
          <tr>
            <td align="center" style={{ padding: "40px 16px" }}>
              <table
                role="presentation"
                width="100%"
                cellPadding={0}
                cellSpacing={0}
                style={{ maxWidth: 560 }}
              >
                <tbody>
                  <tr>
                    <td style={{ padding: "0 0 24px" }}>
                      <img
                        src={`${origin}/email/logo.png`}
                        width={100}
                        height={28}
                        alt="Synnax"
                        style={{ display: "block", border: 0 }}
                      />
                    </td>
                  </tr>
                  <tr>
                    <td
                      style={{
                        padding: 32,
                        background: COLOR.card,
                        border: `1px solid ${COLOR.border}`,
                        borderRadius: 8,
                      }}
                    >
                      <h1
                        style={{
                          margin: "0 0 16px",
                          fontSize: 20,
                          lineHeight: "28px",
                          fontWeight: 600,
                          color: COLOR.heading,
                        }}
                      >
                        {notice.heading}
                      </h1>
                      {notice.paragraphs.map((p) => (
                        <p key={p} style={TEXT}>
                          {p}
                        </p>
                      ))}
                      <Facts facts={notice.facts} />
                      <Button action={notice.action} origin={origin} />
                    </td>
                  </tr>
                  <tr>
                    <td style={{ padding: "24px 0 0" }}>
                      <p
                        style={{
                          ...TEXT,
                          margin: "0 0 4px",
                          fontSize: 12,
                          color: COLOR.muted,
                        }}
                      >
                        {notice.reason}
                      </p>
                      <p
                        style={{ ...TEXT, margin: 0, fontSize: 12, color: COLOR.muted }}
                      >
                        Synnax Labs ·{" "}
                        <a href={origin} style={{ color: COLOR.muted }}>
                          {new URL(origin).host}
                        </a>
                      </p>
                    </td>
                  </tr>
                </tbody>
              </table>
            </td>
          </tr>
        </tbody>
      </table>
    </body>
  </html>
);

const Facts = ({ facts }: { facts: Fact[] }): ReactElement => (
  <table
    role="presentation"
    width="100%"
    cellPadding={0}
    cellSpacing={0}
    style={{
      margin: "8px 0 24px",
      background: COLOR.panel,
      border: `1px solid ${COLOR.border}`,
      borderRadius: 6,
      borderCollapse: "separate",
    }}
  >
    <tbody>
      {facts.map(({ label, value }, i) => {
        const cell: CSSProperties = {
          padding: "10px 16px",
          fontSize: 13,
          lineHeight: "20px",
          borderTop: i === 0 ? "none" : `1px solid ${COLOR.border}`,
        };
        return (
          <tr key={label}>
            <td width="40%" style={{ ...cell, color: COLOR.muted }}>
              {label}
            </td>
            <td style={{ ...cell, color: COLOR.heading, fontWeight: 500 }}>{value}</td>
          </tr>
        );
      })}
    </tbody>
  </table>
);

// Padding sits on the cell as well as the link, so Outlook, which ignores padding on
// a link, still draws the whole button.
const Button = ({
  action,
  origin,
}: {
  action: Action;
  origin: string;
}): ReactElement => (
  <table role="presentation" cellPadding={0} cellSpacing={0}>
    <tbody>
      <tr>
        <td style={{ background: COLOR.primary, borderRadius: 6 }}>
          <a
            href={`${origin}${action.href}`}
            style={{
              display: "inline-block",
              padding: "10px 18px",
              fontSize: 14,
              lineHeight: "20px",
              fontWeight: 600,
              color: "#ffffff",
              textDecoration: "none",
              borderRadius: 6,
            }}
          >
            {action.label}
          </a>
        </td>
      </tr>
    </tbody>
  </table>
);

/** render lays a notice out as HTML and as plain text, with links on `origin`. */
export const render = (notice: Notice, origin: string): Rendered => {
  const text = [
    notice.heading,
    ...notice.paragraphs,
    notice.facts.map(({ label, value }) => `${label}: ${value}`).join("\n"),
    `${notice.action.label}: ${origin}${notice.action.href}`,
    `${notice.reason}\nSynnax Labs, ${origin}`,
  ].join("\n\n");
  const markup = renderToStaticMarkup(<Email notice={notice} origin={origin} />);
  return { text, html: `<!DOCTYPE html>${markup}` };
};

export interface ResendConfig {
  apiKey: string;
  from: string;
  /** origin is the portal's URL, which links and images in the mail point to. */
  origin: string;
}

export const resend = ({ apiKey, from, origin }: ResendConfig): Mailer => {
  const client = new Resend(apiKey);
  return {
    send: async (message) => {
      const { text, html } = render(message, origin);
      const { error } = await client.emails.send({
        from,
        to: message.to,
        subject: message.subject,
        text,
        html,
      });
      if (error != null) throw new Error(`mail: ${error.message}`);
    },
  };
};

/** log prints messages instead of sending them, so previews never mail real people. */
export const log = (): Mailer => ({
  send: async ({ to, subject }) => console.info("mail not sent", { to, subject }),
});

/** memory collects messages instead of sending them. For tests. */
export const memory = (): Mailer & { sent: Message[] } => {
  const sent: Message[] = [];
  return { sent, send: async (message) => void sent.push(message) };
};
