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
import { act, renderHook } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, it } from "vitest";

import { Credential } from "@/credential";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

describe("Password queries", () => {
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  it("should set another user's password", async () => {
    const viewer = await createTestClientWithRole(client, "Viewer");
    await viewer.auth.sessions.retrieve();
    const { subject } = viewer.auth;
    const { username } = viewer.params;
    if (subject == null || username == null) throw new Error("not logged in");
    // ending a live client's session here would make it log a failed login
    await viewer.close();
    const set = renderHook(() => Credential.Password.useSet(), { wrapper });
    await act(async () => {
      await set.result.current.updateAsync({ subject, password: "new" });
    });
    await createTestClient({ username, password: "new" }).auth.sessions.retrieve();
  });
});
