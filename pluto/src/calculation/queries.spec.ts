// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
} from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Calculation } from "@/calculation";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

describe("queries", () => {
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useIsOutput", () => {
    it("should tell an output from a channel that no calculation writes", async () => {
      const calc = await client.calculations.create({
        name: id.create(),
        expression: "return 2",
      });
      const plain = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      const { result } = renderHook(
        () => ({
          output: Calculation.useIsOutput(calc.output),
          plain: Calculation.useIsOutput(plain.key),
        }),
        { wrapper },
      );
      // The client that creates a calculation holds its answer, so only plain loads.
      expect(result.current.plain).toBeUndefined();
      await waitFor(() =>
        expect(result.current).toEqual({ output: true, plain: false }),
      );
    });

    it("should answer false at once for the key 0", () => {
      const { result } = renderHook(() => Calculation.useIsOutput(0), { wrapper });
      expect(result.current).toBe(false);
    });

    it("should count an output whose calculation the reader cannot retrieve", async () => {
      const calc = await client.calculations.create({
        name: id.create(),
        expression: "return 2",
      });
      const reader = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [channel.resourceID(calc.output)],
        actions: ["retrieve"],
      });
      const { result } = renderHook(() => Calculation.useIsOutput(calc.output), {
        wrapper: await createAsyncSynnaxWrapper({ client: reader }),
      });
      await waitFor(() => expect(result.current).toBe(true));
    });

    it("should follow a calculation that is deleted", async () => {
      const calc = await client.calculations.create({
        name: id.create(),
        expression: "return 2",
      });
      const { result } = renderHook(() => Calculation.useIsOutput(calc.output), {
        wrapper,
      });
      await waitFor(() => expect(result.current).toBe(true));
      await client.calculations.delete(calc.key);
      await waitFor(() => expect(result.current).toBe(false));
    });
  });
});
