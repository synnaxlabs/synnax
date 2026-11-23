// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Mock, vi } from "vitest";

import { Portal } from "@/platform/portal";

/** The origin of every portal client a spec creates. */
export const TEST_URL = "https://portal.test";

type Answer = (url: string, init?: RequestInit) => Promise<Response>;

const unreachable: Answer = async () => {
  throw new TypeError("Failed to fetch");
};

export interface TestPortal {
  client: Portal.Client;
  /** Records each request the client sends. */
  fetch: Mock<Answer>;
}

/**
 * Creates a real portal client at {@link TEST_URL} whose requests `answer` serves. By
 * default the portal cannot be reached, as on a machine with no internet access.
 */
export const createTestPortal = (answer: Answer = unreachable): TestPortal => {
  const fetch = vi.fn(answer);
  const client = new Portal.Client({
    url: TEST_URL,
    fetch: async (input, init) =>
      await fetch(input instanceof Request ? input.url : input.toString(), init),
  });
  return { client, fetch };
};
