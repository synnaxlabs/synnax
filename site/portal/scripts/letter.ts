// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Previews, tests, and sends the release notes letter of a version:
//
//   pnpm --filter @synnaxlabs/portal letter 0.61.0 preview
//   RESEND_API_KEY=... pnpm --filter @synnaxlabs/portal letter 0.61.0 test
//   RESEND_API_KEY=... pnpm --filter @synnaxlabs/portal letter 0.61.0 send
//
// preview writes the HTML to a temporary file. test mails the letter to the sender
// alone. send mails it to everyone on the release notes list, one time per version.

import { writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

import { Resend } from "resend";

import images from "../src/letters/images.json" with { type: "json" };
import { LETTERS } from "../src/letters/index.ts";
import { resend } from "../src/server/audience.ts";
import { render, UNSUBSCRIBE } from "../src/server/letter.tsx";

const SENDER = "ebonilla@synnaxlabs.com";
const FROM = `Emiliano Bonilla <${SENDER}>`;
const ORIGIN = "https://portal.synnaxlabs.com";
const MODES = ["preview", "test", "send"];

const [version, mode] = process.argv.slice(2);
const letter = LETTERS.find((l) => l.version === version);
if (letter == null || !MODES.includes(mode))
  throw new Error(
    `usage: letter <${LETTERS.map((l) => l.version).join("|")}> <${MODES.join("|")}>`,
  );
const { html, text } = render(letter, ORIGIN, images);

const apiKey = (): string => {
  const key = process.env.RESEND_API_KEY;
  if (key == null || key === "") throw new Error("RESEND_API_KEY is not set");
  return key;
};

if (mode === "preview") {
  const file = path.join(tmpdir(), `letter-${version}.html`);
  await writeFile(file, html);
  console.info(file);
} else if (mode === "test") {
  const { error } = await new Resend(apiKey()).emails.send({
    from: FROM,
    to: SENDER,
    subject: `[Test] ${letter.subject}`,
    html: html.replaceAll(UNSUBSCRIBE, ORIGIN),
    text: text.replaceAll(UNSUBSCRIBE, ORIGIN),
  });
  if (error != null) throw new Error(`letter: ${error.message}`);
  console.info(`sent a test to ${SENDER}`);
} else {
  await resend(apiKey()).send({
    name: `v${letter.version}`,
    from: FROM,
    subject: letter.subject,
    html,
    text,
  });
  console.info(`sent v${letter.version} to the release notes list`);
}
