// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { calculation, channel, DataType } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
  holdFailures,
  holdResponses,
  rejectRequests,
} from "@synnaxlabs/client/testutil";
import { Status } from "@synnaxlabs/lyra/status";
import { id, TimeStamp } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it, onTestFinished } from "vitest";

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

  describe("useNonOutputFilter", () => {
    const LOOKUP = "/calculation/retrieve";

    const createPlain = async () =>
      await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });

    const renderFilter = (open = true) =>
      renderHook(
        () => ({
          stored: Calculation.useNonOutputFilter({ open }),
          reported: Status.useNotifications().statuses,
        }),
        { wrapper },
      );

    it("should keep only the channels that no calculation writes, once loaded", async () => {
      const calc = await client.calculations.create({
        name: id.create(),
        expression: "return 2",
      });
      const plain = await createPlain();
      const output = await client.channels.retrieve(calc.output);
      const { result } = renderFilter();
      // A channel whose answer has not loaded stays out.
      expect(result.current.stored.filter(plain)).toBe(false);
      await waitFor(() => expect(result.current.stored.filter(plain)).toBe(true));
      await waitFor(() => expect(result.current.stored.filter(output)).toBe(false));
      await waitFor(() => expect(result.current.stored.status).toBeUndefined());
    });

    it("should show a loading status while a lookup runs, and send it once", async () => {
      const plain = await createPlain();
      const { responded, release, sent } = holdResponses(client, LOOKUP);
      const { result } = renderFilter();
      expect(result.current.stored.filter(plain)).toBe(false);
      await waitFor(() =>
        expect(result.current.stored.status?.variant).toBe("loading"),
      );
      await responded;
      expect(result.current.stored.filter(plain)).toBe(false);
      await waitFor(() => expect(result.current.stored.filter(plain)).toBe(false));
      expect(sent()).toBe(1);
      release();
      await waitFor(() => expect(result.current.stored.filter(plain)).toBe(true));
      await waitFor(() => expect(result.current.stored.status).toBeUndefined());
    });

    it("should report a failure once while open, until a later lookup answers", async () => {
      const first = await createPlain();
      const second = await createPlain();
      let rejected = 0;
      const failing = rejectRequests(
        client,
        (target) => target === LOOKUP && ++rejected > 0,
        new Error("calculations unavailable"),
      );
      onTestFinished(() => failing.mockRestore());
      const { result } = renderFilter();
      result.current.stored.filter(first);
      await waitFor(() => expect(result.current.stored.status?.variant).toBe("error"));
      expect(result.current.stored.status?.message).toEqual("Failed to load channels");
      expect(result.current.stored.status?.description).toContain(
        "Synnax could not check which channels a calculation writes. Check the " +
          "connection to the Core, then open the picker again.",
      );
      expect(result.current.stored.status?.description).toContain(
        "calculations unavailable",
      );
      // A second failure in the same episode is not reported again.
      result.current.stored.filter(second);
      await waitFor(() => expect(rejected).toBe(2));
      await waitFor(() => expect(result.current.stored.status?.variant).toBe("error"));
      expect(
        result.current.reported.map(({ message, count }) => ({ message, count })),
      ).toEqual([{ message: "Failed to load channels", count: 1 }]);
      failing.mockRestore();
      // The channels without an answer join the next lookup, which clears the failure.
      const { release } = holdResponses(client, LOOKUP);
      result.current.stored.filter(first);
      result.current.stored.filter(second);
      await waitFor(() =>
        expect(result.current.stored.status?.variant).toBe("loading"),
      );
      release();
      await waitFor(() => expect(result.current.stored.filter(first)).toBe(true));
      await waitFor(() => expect(result.current.stored.status).toBeUndefined());
    });

    it("should report no failure that lands after the picker closes", async () => {
      const plain = await createPlain();
      const { release } = holdFailures(
        client,
        LOOKUP,
        new Error("calculations unavailable"),
      );
      const { result, rerender } = renderHook(
        ({ open }) => ({
          stored: Calculation.useNonOutputFilter({ open }),
          reported: Status.useNotifications().statuses,
        }),
        { wrapper, initialProps: { open: true } },
      );
      result.current.stored.filter(plain);
      await waitFor(() =>
        expect(result.current.stored.status?.variant).toBe("loading"),
      );
      rerender({ open: false });
      release();
      await waitFor(() => expect(result.current.stored.status?.variant).toBe("error"));
      expect(result.current.reported).toEqual([]);
    });

    it("should report no failure while closed, and still show it", async () => {
      const plain = await createPlain();
      const failing = rejectRequests(
        client,
        (target) => target === LOOKUP,
        new Error("calculations unavailable"),
      );
      onTestFinished(() => failing.mockRestore());
      const { result } = renderFilter(false);
      result.current.stored.filter(plain);
      await waitFor(() => expect(result.current.stored.status?.variant).toBe("error"));
      expect(result.current.reported).toEqual([]);
    });
  });

  describe("useResultStatus", () => {
    const setStatus = async (
      calc: calculation.Calculation,
      message: string,
      description: string = "",
    ): Promise<void> => {
      await client.statuses.set({
        owner: calculation.resourceID(calc.key),
        name: "Calculation Status",
        variant: "error",
        message,
        description,
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
      expect(result.current.data?.variant).toEqual("error");
    });

    it("should answer with null for a channel that no calculation writes", async () => {
      const plain = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      const { result } = renderHook(
        () => Calculation.useResultStatus({ channel: plain.key }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toBeNull();
    });

    it("should answer with null for a calculation with no status", async () => {
      const calc = await client.calculations.create({
        name: id.create(),
        expression: "return 2",
      });
      const { result } = renderHook(
        () => Calculation.useResultStatus({ channel: calc.output }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toBeNull();
    });

    it("should follow a change of the description alone", async () => {
      const calc = await client.calculations.create({
        name: id.create(),
        expression: "return 2",
      });
      await setStatus(calc, "Failed to run calculation", "cannot read a");
      const { result } = renderHook(
        () => Calculation.useResultStatus({ channel: calc.output }),
        { wrapper },
      );
      await waitFor(() =>
        expect(result.current.data?.description).toEqual("cannot read a"),
      );
      await act(
        async () => await setStatus(calc, "Failed to run calculation", "cannot read b"),
      );
      await waitFor(() =>
        expect(result.current.data?.description).toEqual("cannot read b"),
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
