// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { user } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithRole,
} from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import {
  useCreateAPIKey,
  useDeleteAPIKey,
  useResultAPIKeys,
  useResultSessions,
  useSetPassword,
} from "@/credential/queries";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

describe("Credential queries", () => {
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const loginAsViewer = async () => {
    const viewer = await createTestClientWithRole(client, "Viewer");
    await viewer.auth.sessions.retrieve();
    const { subject } = viewer.auth;
    const { username } = viewer.params;
    if (subject == null || username == null) throw new Error("not logged in");
    return { viewer, subject, username };
  };

  it("should create a key, list it, and delete it", async () => {
    const u = await client.users.create({ name: `credential-${id.create()}` });
    const subject = user.ontologyID(u.key);
    const create = renderHook(() => useCreateAPIKey(), { wrapper });
    await act(async () => {
      await create.result.current.updateAsync({ subject, name: "ci" });
    });
    const keys = renderHook(() => useResultAPIKeys({ subject }), { wrapper });
    await waitFor(() => expect(keys.result.current.data).toHaveLength(1));
    const [key] = keys.result.current.data ?? [];
    expect(key.name).toEqual("ci");
    const del = renderHook(() => useDeleteAPIKey(), { wrapper });
    await act(async () => {
      await del.result.current.updateAsync(key.key);
    });
    expect(await client.auth.apiKeys.retrieve([subject])).toHaveLength(0);
  });

  it("should list a subject's sessions", async () => {
    const { subject } = await loginAsViewer();
    const sessions = renderHook(() => useResultSessions({ subject }), { wrapper });
    await waitFor(() => expect(sessions.result.current.data).toHaveLength(1));
  });

  it("should set another user's password", async () => {
    const { viewer, subject, username } = await loginAsViewer();
    // ending a live client's session here would make it log a failed login
    await viewer.close();
    const set = renderHook(() => useSetPassword(), { wrapper });
    await act(async () => {
      await set.result.current.updateAsync({ subject, password: "new" });
    });
    await createTestClient({ username, password: "new" }).auth.sessions.retrieve();
  });
});
