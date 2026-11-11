// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";

import { Button, COLOR, Frame, HEADING, type Rendered, TEXT } from "@/server/mail";

/** Feature is one change that a letter shows, with a picture of it. */
export interface Feature {
  title: string;
  /** text says what the change does, in one or two sentences. */
  text: string;
  /** image is the Studio id of the still or video that the picture comes from. */
  image: string;
}

/** Letter is the release notes email of one release. */
export interface Letter {
  /** version is the release, such as "0.61.0". */
  version: string;
  subject: string;
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

/** notes returns the URL of the release notes of `version`. */
const notes = (version: string): string =>
  `https://docs.synnaxlabs.com/releases#${version.replaceAll(".", "-")}`;

// Studio makes mail copies at 16:10, at twice this size.
const IMAGE = { width: 496, height: 310 };

const Body = ({
  letter,
  images,
}: {
  letter: Letter;
  images: string[];
}): ReactElement => (
  <>
    <h1 style={HEADING}>Synnax v{letter.version.split(".").slice(0, 2).join(".")}</h1>
    {letter.opening.map((p) => (
      <p key={p} style={TEXT}>
        {p}
      </p>
    ))}
    {letter.features.map(({ title, text }, i) => (
      <div key={title} style={{ margin: "32px 0 0" }}>
        <img
          src={images[i]}
          width={IMAGE.width}
          height={IMAGE.height}
          alt={title}
          style={{
            display: "block",
            width: "100%",
            maxWidth: IMAGE.width,
            height: "auto",
            border: `1px solid ${COLOR.border}`,
            borderRadius: 6,
          }}
        />
        <h2
          style={{
            margin: "20px 0 8px",
            fontSize: 16,
            lineHeight: "24px",
            fontWeight: 600,
            color: COLOR.heading,
          }}
        >
          {title}
        </h2>
        <p style={TEXT}>{text}</p>
      </div>
    ))}
    {letter.breaking != null && (
      <p
        style={{
          ...TEXT,
          margin: "16px 0 24px",
          padding: "12px 16px",
          background: COLOR.panel,
          border: `1px solid ${COLOR.border}`,
          borderRadius: 6,
        }}
      >
        <strong style={{ color: COLOR.heading }}>Breaking changes.</strong>{" "}
        {letter.breaking}
      </p>
    )}
    <div style={{ margin: "8px 0 24px" }}>
      <Button label="Read the release notes" href={notes(letter.version)} />
    </div>
    {letter.closing.map((p, i) => (
      <p
        key={p}
        style={{ ...TEXT, margin: i === letter.closing.length - 1 ? 0 : TEXT.margin }}
      >
        {p}
      </p>
    ))}
  </>
);

/**
 * render lays a letter out as HTML and as plain text. `images` maps each Studio id to
 * the URL of its mail copy. Throws when a feature's image has no mail copy.
 */
export const render = (
  letter: Letter,
  origin: string,
  images: Record<string, string>,
): Rendered => {
  const urls = letter.features.map(({ image }) => {
    const url = images[image];
    if (url == null)
      throw new Error(`letter: ${image} has no mail copy. Run pnpm mail ${image}`);
    return url;
  });
  const text = [
    ...letter.opening,
    ...letter.features.map(({ title, text }) => `${title}\n${text}`),
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
    <Frame
      subject={letter.subject}
      preview={letter.opening[0]}
      origin={origin}
      reason={reason}
    >
      <Body letter={letter} images={urls} />
    </Frame>,
  );
  return { text, html: `<!DOCTYPE html>${markup}` };
};
