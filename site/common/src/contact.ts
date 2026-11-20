// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** The Formspree form that receives the contact messages of every site. */
export const ENDPOINT = "https://formspree.io/f/mgegebvq";

/**
 * Sends a contact message.
 * @returns true when Formspree accepts the message, and false when it refuses it or
 * the network fails.
 */
export const send = async (data: FormData): Promise<boolean> => {
  try {
    const res = await fetch(ENDPOINT, {
      method: "POST",
      body: data,
      headers: { Accept: "application/json" },
    });
    return res.ok;
  } catch {
    return false;
  }
};
