// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { enumeration } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { color, id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Options } from "@/schematic/node/common/options";
import { createAsyncSynnaxWrapper } from "@/testutil";

const client = createTestClient();
const remote = createTestClient();

describe("Options.use", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  it("should return no states for a node that names no enumeration", () => {
    const { result } = renderHook(() => Options.use(undefined), { wrapper });
    expect(result.current).toEqual([]);
  });

  it("should return the entries of the enumeration", async () => {
    const created = await client.enumerations.create({
      name: `options-${id.create()}`,
      entries: [
        { key: "off", value: 0, name: "Off" },
        {
          key: "on",
          value: 1,
          name: "On",
          color: { type: "fixed", value: color.construct("#FF0000") },
        },
      ],
    });
    const { result } = renderHook(() => Options.use(created.key), { wrapper });
    await waitFor(() =>
      expect(result.current).toEqual([
        { key: "off", name: "Off", value: 0, color: undefined },
        { key: "on", name: "On", value: 1, color: color.construct("#FF0000") },
      ]),
    );
    await act(async () => {
      await remote.enumerations.retrieve(created.key);
      await remote.enumerations.dispatch(
        created.key,
        enumeration.set({ path: ["entries", "on", "name"], value: "Running" }),
      );
    });
    await waitFor(() => expect(result.current[1].name).toBe("Running"));
  });
});
