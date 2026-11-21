// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { calculation, DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id, TimeStamp } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
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
      await waitFor(() => expect(result.current.output).toBe(true));
      expect(result.current.plain).toBe(false);
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

  describe("useNonOutputFilter", () => {
    it("should keep only the channels that no calculation writes, once loaded", async () => {
      const calc = await client.calculations.create({
        name: id.create(),
        expression: "return 2",
      });
      const plain = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      const output = await client.channels.retrieve(calc.output);
      const { result } = renderHook(() => Calculation.useNonOutputFilter(), {
        wrapper,
      });
      // A channel whose answer has not loaded stays out.
      expect(result.current(plain)).toBe(false);
      await waitFor(() => expect(result.current(plain)).toBe(true));
      await waitFor(() => expect(result.current(output)).toBe(false));
    });
  });

  describe("useResultStatus", () => {
    const setStatus = async (
      calc: calculation.Calculation,
      message: string,
    ): Promise<void> => {
      await client.statuses.set({
        owner: calculation.resourceID(calc.key),
        name: "Calculation Status",
        variant: "error",
        message,
        time: TimeStamp.now(),
      });
    };

    it("should answer with the status of the calculation that writes a channel", async () => {
      const calc = await client.calculations.create({
        name: id.create(),
        expression: "return 2",
      });
      await setStatus(calc, "invalid expression");
      const { result } = renderHook(
        () => Calculation.useResultStatus({ channel: calc.output }),
        { wrapper },
      );
      await waitFor(() =>
        expect(result.current.data?.message).toEqual("invalid expression"),
      );
    });

    it("should follow a change of the status", async () => {
      const calc = await client.calculations.create({
        name: id.create(),
        expression: "return 2",
      });
      await setStatus(calc, "invalid expression");
      const { result } = renderHook(
        () => Calculation.useResultStatus({ channel: calc.output }),
        { wrapper },
      );
      await waitFor(() =>
        expect(result.current.data?.message).toEqual("invalid expression"),
      );
      await act(async () => await setStatus(calc, "broken expression"));
      await waitFor(() =>
        expect(result.current.data?.message).toEqual("broken expression"),
      );
    });
  });
});
