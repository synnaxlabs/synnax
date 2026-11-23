// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { user } from "@synnaxlabs/client";
import { createEmptyRole, createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Credential } from "@/credential";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

describe("OIDC invite queries", () => {
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  it("should retrieve no invites for a user without one", async () => {
    const { key: role } = await createEmptyRole(client);
    const created = await client.users.create({ name: "No invite", role });
    const subject = user.resourceID(created.key);
    const { result } = renderHook(() => Credential.OIDC.Invite.useResult({ subject }), {
      wrapper,
    });
    await waitFor(() => expect(result.current.data).toEqual([]));
  });

  it("should report an invite the Core refuses", async () => {
    const { result } = renderHook(() => Credential.OIDC.Invite.useCreate(), {
      wrapper,
    });
    let created: boolean | undefined;
    await act(async () => {
      created = await result.current.updateAsync(uuid.create());
    });
    expect(created).toBe(false);
    expect(result.current.status.message).toEqual("Failed to create invite");
  });

  it("should revoke an unknown invite", async () => {
    const { result } = renderHook(() => Credential.OIDC.Invite.useRevoke(), {
      wrapper,
    });
    let revoked: boolean | undefined;
    await act(async () => {
      revoked = await result.current.updateAsync(uuid.create());
    });
    expect(revoked).toBe(true);
  });
});
