// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { user } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Credential } from "@/credential";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

describe("OIDC queries", () => {
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  it("should retrieve no links for a user without any", async () => {
    const created = await client.users.create({ name: "No links" });
    const subject = user.ontologyID(created.key);
    const { result } = renderHook(() => Credential.OIDC.useResult({ subject }), {
      wrapper,
    });
    await waitFor(() => expect(result.current.data).toEqual([]));
  });

  it("should report a link the Core refuses", async () => {
    const { result } = renderHook(() => Credential.OIDC.useLink(), { wrapper });
    let linked: boolean | undefined;
    await act(async () => {
      linked = await result.current.updateAsync({
        provider: "idp",
        code: "code",
        verifier: "verifier",
        redirectUri: "http://127.0.0.1:4242/callback",
      });
    });
    expect(linked).toBe(false);
    expect(result.current.status.message).toEqual("Failed to link provider account");
  });
});
