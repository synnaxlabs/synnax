// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { enumeration, schematic } from "@synnaxlabs/client";
import { createTestClient, waitForStreamLive } from "@synnaxlabs/client/testutil";
import { color, id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Options } from "@/schematic/node/common/options";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const remote = createTestClient();

const OPTIONS = schematic.stateMappingZ
  .array()
  .parse([{ key: "own", name: "Own", value: 7 }]);

describe("Options.use", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    await waitForStreamLive(client.connection);
  });

  it("should return the options of a node that names no enumeration", () => {
    const { result } = renderHook(() => Options.use({ options: OPTIONS }), { wrapper });
    expect(result.current).toBe(OPTIONS);
  });

  it("should return the entries of the enumeration in place of the options", async () => {
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
    const { result } = renderHook(
      () => Options.use({ enumeration: created.key, options: OPTIONS }),
      { wrapper },
    );
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
