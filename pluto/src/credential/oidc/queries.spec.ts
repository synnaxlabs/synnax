// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  AccountLinkedError,
  type auth,
  ProviderLinkedError,
  user,
} from "@synnaxlabs/client";
import { createEmptyRole, createTestClient } from "@synnaxlabs/client/testutil";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Credential } from "@/credential";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

describe("OIDC queries", () => {
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  it("should retrieve no links for a user without any", async () => {
    const { key: role } = await createEmptyRole(client);
    const created = await client.users.create({ name: "No links", role });
    const subject = user.resourceID(created.key);
    const { result } = renderHook(() => Credential.OIDC.useResult({ subject }), {
      wrapper,
    });
    await waitFor(() => expect(result.current.data).toEqual([]));
  });

  const CREDENTIALS: auth.OIDCCredentials = {
    provider: "Google",
    code: "code",
    verifier: "verifier",
    redirectUri: "http://127.0.0.1:4242/callback",
  };

  const link = async () => {
    const { result } = renderHook(() => Credential.OIDC.useLink(), { wrapper });
    let linked: boolean | undefined;
    await act(async () => {
      linked = await result.current.updateAsync(CREDENTIALS);
    });
    expect(linked).toBe(false);
    return result.current.status;
  };

  it("should report a link the Core refuses", async () => {
    expect((await link()).message).toEqual("Failed to link account");
  });

  it.each([
    [new ProviderLinkedError("linked"), "You already have a Google account linked."],
    [
      new AccountLinkedError("linked"),
      "This Google account is linked to another user.",
    ],
  ])("should name the provider when the Core refuses %s", async (err, description) => {
    vi.spyOn(client.auth.oidc, "link").mockRejectedValueOnce(err);
    const status = await link();
    expect(status.message).toEqual("Failed to link account");
    expect(status.description).toEqual(description);
  });
});
