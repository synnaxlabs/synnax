// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Resend } from "resend";

export interface Message {
  to: string[];
  subject: string;
  text: string;
}

/** Mailer sends the transactional mail Clerk does not: expiry and revocation notices. */
export interface Mailer {
  send: (message: Message) => Promise<void>;
}

export const resend = (apiKey: string, from: string): Mailer => {
  const client = new Resend(apiKey);
  return {
    send: async ({ to, subject, text }) => {
      const { error } = await client.emails.send({ from, to, subject, text });
      if (error != null) throw new Error(`mail: ${error.message}`);
    },
  };
};

/** memory collects messages instead of sending them. For tests. */
export const memory = (): Mailer & { sent: Message[] } => {
  const sent: Message[] = [];
  return { sent, send: async (message) => void sent.push(message) };
};
