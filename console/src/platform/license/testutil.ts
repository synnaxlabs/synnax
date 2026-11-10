// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Mock, onTestFinished, vi } from "vitest";

import { HANDOFF_URL } from "@/platform/license/urls";

const PORTAL_ORIGIN = new URL(HANDOFF_URL).origin;

type Answer = (url: string, init?: RequestInit) => Promise<Response>;

/**
 * Answers every request to the portal with `answer`, and sends every other request to
 * the real fetch. Restores fetch when the test finishes.
 *
 * @returns a spy that records only the requests to the portal.
 */
export const stubHub = (answer: Answer): Mock<Answer> => {
  const hub = vi.fn(answer);
  const real = globalThis.fetch;
  const spy = vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
    const url = input instanceof Request ? input.url : input.toString();
    if (new URL(url).origin === PORTAL_ORIGIN) return await hub(url, init);
    return await real(input, init);
  });
  onTestFinished(() => spy.mockRestore());
  return hub;
};
