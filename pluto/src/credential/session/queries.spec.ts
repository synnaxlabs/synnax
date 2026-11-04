// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createTestClient,
  createTestClientWithRole,
} from "@synnaxlabs/client/testutil";
import { renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Credential } from "@/credential";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

describe("Session queries", () => {
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  it("should list a subject's sessions", async () => {
    const viewer = await createTestClientWithRole(client, "Viewer");
    await viewer.auth.sessions.retrieve();
    const { subject } = viewer.auth;
    if (subject == null) throw new Error("not logged in");
    const sessions = renderHook(() => Credential.Session.useResult({ subject }), {
      wrapper,
    });
    await waitFor(() => expect(sessions.result.current.data).toHaveLength(1));
  });
});
