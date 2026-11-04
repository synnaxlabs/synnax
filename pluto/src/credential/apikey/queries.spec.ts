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
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Credential } from "@/credential";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

describe("API key queries", () => {
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  it("should create a key, list it, and delete it", async () => {
    const u = await client.users.create({ name: `credential-${id.create()}` });
    const subject = user.ontologyID(u.key);
    const create = renderHook(() => Credential.APIKey.useCreate(), { wrapper });
    await act(async () => {
      await create.result.current.updateAsync({ subject, name: "ci" });
    });
    const keys = renderHook(() => Credential.APIKey.useResult({ subject }), {
      wrapper,
    });
    await waitFor(() => expect(keys.result.current.data).toHaveLength(1));
    const [key] = keys.result.current.data ?? [];
    expect(key.name).toEqual("ci");
    const del = renderHook(() => Credential.APIKey.useDelete(), { wrapper });
    await act(async () => {
      await del.result.current.updateAsync(key.key);
    });
    expect(await client.auth.apiKeys.retrieve([subject])).toHaveLength(0);
  });
});
