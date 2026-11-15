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

import { Button, COLOR, Footer, Page, type Rendered } from "@/server/mail";

/** Feature is one change that a letter shows, with a shot of it. */
export interface Feature {
  /** area is the part of Synnax that the change is in, such as "Arc". */
  area: string;
  title: string;
  /** text says what the change does, in one or two sentences. */
  text: string;
  /** href is the path of the change's page on the docs site. */
  href: string;
  /** image is the Studio id of the video that the shot comes from. */
  image: string;
}

/** Letter is the release notes email of one release. */
export interface Letter {
  /** version is the release, such as "0.61.0". */
  version: string;
  subject: string;
  /** title is the headline of the masthead. */
  title: string;
  /** hero is the Studio id of the still under the title. */
  hero: string;
  /** opening is the paragraphs above the features. Inboxes preview the first one. */
  opening: string[];
  features: Feature[];
  /** breaking tells what to change before an upgrade, or is null for nothing. */
  breaking: string | null;
  /** closing is the paragraphs under the button. The last one is the signature. */
  closing: string[];
}

/** UNSUBSCRIBE is the placeholder that Resend fills with each reader's link. */
export const UNSUBSCRIBE = "{{{RESEND_UNSUBSCRIBE_URL}}}";

const REASON =
  "You get this email because you have a Synnax account or asked for our release " +
  "notes.";

const DOCS = "https://docs.synnaxlabs.com";

/** notes returns the URL of the release notes of `version`. */
const notes = (version: string): string =>
  `${DOCS}/releases#${version.replaceAll(".", "-")}`;

// Studio makes the mail copies at twice these sizes.
const HERO = { width: 576, height: 320 };
const SHOT = { width: 512, height: 288 };

const BAND = { background: "#0f1117", text: "#f4f5f8", label: "#9aa2b4" };

const MONO = "'SF Mono', Menlo, Consolas, 'Liberation Mono', monospace";

const P: CSSProperties = {
  margin: "0 0 16px",
  fontSize: 16,
  lineHeight: "26px",
  color: COLOR.text,
};

const LABEL: CSSProperties = {
  margin: "0 0 8px",
  fontFamily: MONO,
  fontSize: 12,
  lineHeight: "16px",
  letterSpacing: "0.06em",
  textTransform: "uppercase",
  color: COLOR.primary,
};

const SECTION: CSSProperties = {
  margin: "36px 0 0",
  paddingTop: 32,
  borderTop: `1px solid ${COLOR.border}`,
};

interface ImageProps {
  src: string;
  size: { width: number; height: number };
  alt: string;
  style?: CSSProperties;
}

const Image = ({ src, size, alt, style }: ImageProps): ReactElement => (
  <img
    src={src}
    width={size.width}
    height={size.height}
    alt={alt}
    style={{
      display: "block",
      width: "100%",
      maxWidth: size.width,
      height: "auto",
      border: 0,
      ...style,
    }}
  />
);

const Masthead = ({
  letter,
  origin,
  hero,
}: {
  letter: Letter;
  origin: string;
  hero: string;
}): ReactElement => (
  <tr>
    <td>
      <table
        role="presentation"
        width="100%"
        cellPadding={0}
        cellSpacing={0}
        style={{ background: BAND.background, borderRadius: 14, overflow: "hidden" }}
      >
        <tbody>
          <tr>
            <td style={{ padding: "28px 32px 28px" }}>
              <img
                src={`${origin}/email/logo-white.png`}
                width={100}
                height={28}
                alt="Synnax"
                style={{ display: "block", border: 0, margin: "0 0 40px" }}
              />
              <p style={{ ...LABEL, color: BAND.label, margin: "0 0 12px" }}>
                Release {letter.version.split(".").slice(0, 2).join(".")}
              </p>
              <h1
                style={{
                  margin: 0,
                  fontSize: 30,
                  lineHeight: "38px",
                  fontWeight: 500,
                  letterSpacing: "-0.01em",
                  color: BAND.text,
                }}
              >
                {letter.title}
              </h1>
            </td>
          </tr>
          <tr>
            <td>
              <Image src={hero} size={HERO} alt={letter.title} />
            </td>
          </tr>
        </tbody>
      </table>
    </td>
  </tr>
);

const Body = ({ letter, shots }: { letter: Letter; shots: string[] }): ReactElement => (
  <tr>
    <td style={{ paddingTop: 12 }}>
      <table
        role="presentation"
        width="100%"
        cellPadding={0}
        cellSpacing={0}
        style={{ background: COLOR.card, borderRadius: 14 }}
      >
        <tbody>
          <tr>
            <td style={{ padding: "36px 32px 40px" }}>
              {letter.opening.map((p) => (
                <p key={p} style={P}>
                  {p}
                </p>
              ))}
              {letter.features.map(({ area, title, text, href }, i) => (
                <div key={title} style={SECTION}>
                  <p style={LABEL}>{area}</p>
                  <h2
                    style={{
                      margin: "0 0 10px",
                      fontSize: 20,
                      lineHeight: "28px",
                      fontWeight: 600,
                      color: COLOR.heading,
                    }}
                  >
                    <a
                      href={`${DOCS}${href}`}
                      style={{ color: COLOR.heading, textDecoration: "none" }}
                    >
                      {title}
                    </a>
                  </h2>
                  <p style={P}>{text}</p>
                  <Image
                    src={shots[i]}
                    size={SHOT}
                    alt={title}
                    style={{ borderRadius: 8, border: `1px solid ${COLOR.border}` }}
                  />
                </div>
              ))}
              {letter.breaking != null && (
                <div style={SECTION}>
                  <p style={{ ...LABEL, color: "#c2410c" }}>Breaking changes</p>
                  <p style={P}>{letter.breaking}</p>
                </div>
              )}
              <div style={{ margin: "8px 0 0" }}>
                <Button label="Read the release notes" href={notes(letter.version)} />
              </div>
              {letter.closing.map((p, i) => (
                <p
                  key={p}
                  style={
                    i === letter.closing.length - 1
                      ? { ...P, margin: 0, color: COLOR.heading }
                      : { ...P, margin: i === 0 ? "32px 0 4px" : P.margin }
                  }
                >
                  {p}
                </p>
              ))}
            </td>
          </tr>
        </tbody>
      </table>
    </td>
  </tr>
);

/**
 * render lays a letter out as HTML and as plain text. `images` maps each Studio id to
 * the URL of its mail copy. Throws when the hero or a feature has no mail copy.
 */
export const render = (
  letter: Letter,
  origin: string,
  images: Record<string, string>,
): Rendered => {
  const copy = (id: string): string => {
    const url = images[id];
    if (url == null)
      throw new Error(`letter: ${id} has no mail copy. Run pnpm mail ${id}`);
    return url;
  };
  const hero = copy(letter.hero);
  const shots = letter.features.map(({ image }) => copy(image));
  const text = [
    letter.title,
    ...letter.opening,
    ...letter.features.map(
      ({ area, title, text, href }) => `${area}: ${title}\n${text}\n${DOCS}${href}`,
    ),
    ...(letter.breaking != null ? [`Breaking changes. ${letter.breaking}`] : []),
    `Read the release notes: ${notes(letter.version)}`,
    ...letter.closing,
    `${REASON}\nUnsubscribe: ${UNSUBSCRIBE}\nSynnax Labs, ${origin}`,
  ].join("\n\n");
  const reason = (
    <>
      {REASON}{" "}
      <a href={UNSUBSCRIBE} style={{ color: COLOR.muted }}>
        Unsubscribe
      </a>
    </>
  );
  const markup = renderToStaticMarkup(
    <Page
      subject={letter.subject}
      preview={letter.opening[0]}
      width={HERO.width}
      padding="24px 12px 40px"
    >
      <Masthead letter={letter} origin={origin} hero={hero} />
      <Body letter={letter} shots={shots} />
      <Footer origin={origin} reason={reason} />
    </Page>,
  );
  return { text, html: `<!DOCTYPE html>${markup}` };
};
