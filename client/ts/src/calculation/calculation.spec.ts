// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { calculation } from "@/calculation";
import { channel } from "@/channel";
import { PathError } from "@/index";
import { createTestClient, createTestClientWithPolicy, spyOnSend } from "@/testutil";

const client = createTestClient();

const createVirtual = async (): Promise<channel.Channel> =>
  await client.channels.create({
    name: `calc_input_${id.create()}`,
    dataType: DataType.FLOAT32,
    virtual: true,
  });

describe("Calculation", () => {
  describe("create", () => {
    it("should create a calculation on a virtual channel", async () => {
      const output = await createVirtual();
      const c = await client.calculations.create({
        output: output.key,
        expression: "return 2",
      });
      expect(c.output).toEqual(output.key);
      expect(await client.calculations.retrieve({ key: c.key })).toEqual(c);
    });

    it("should make a virtual output named by the create", async () => {
      const input = await createVirtual();
      const name = `calc_output_${id.create()}`;
      const c = await client.calculations.create({
        name,
        expression: `return ${input.name} * 2`,
      });
      const output = await client.channels.retrieve(c.output);
      expect(output.name).toEqual(name);
      expect(output.virtual).toBe(true);
      expect(output.dataType.equals(DataType.FLOAT32)).toBe(true);
      expect(c.reads).toEqual([input.key]);
    });

    it("should refuse a create that names an output and a name", async () => {
      const output = await createVirtual();
      await expect(
        client.calculations.create({
          output: output.key,
          name: `calc_output_${id.create()}`,
          expression: "return 2",
        }),
      ).rejects.toThrow(PathError);
    });
  });

  describe("retrieve", () => {
    it("should retrieve the calculation that writes an output", async () => {
      const c = await client.calculations.create({
        name: `calc_output_${id.create()}`,
        expression: "return 2",
      });
      const found = await client.calculations.retrieve({ output: c.output });
      expect(found).toEqual([c]);
    });

    it("should retrieve no calculation for a channel that none writes", async () => {
      const plain = await createVirtual();
      expect(await client.calculations.retrieve({ output: plain.key })).toEqual([]);
    });
  });

  describe("outputs", () => {
    it("should answer the lookups of a tick in one request", async () => {
      const c = await client.calculations.create({
        name: `calc_output_${id.create()}`,
        expression: "return 2",
      });
      const plain = await createVirtual();
      const local = createTestClient();
      const send = spyOnSend(local);
      const [[written], [unwritten]] = await Promise.all([
        local.calculations.outputs.retrieve([c.output]),
        local.calculations.outputs.retrieve([plain.key]),
      ]);
      expect(written).toEqual({ key: c.output, hidden: false, calculation: c.key });
      expect(unwritten).toEqual({ key: plain.key, hidden: false, calculation: null });
      expect(
        send.mock.calls.filter(([target]) => target === "/calculation/retrieve"),
      ).toHaveLength(1);
    });

    it("should map a channel to a calculation that another client creates", async () => {
      const plain = await createVirtual();
      const local = createTestClient();
      await local.connect();
      await local.calculations.outputs.retrieve([plain.key]);
      const c = await client.calculations.create({
        output: plain.key,
        expression: "return 2",
      });
      await expect
        .poll(() => local.calculations.outputs.get(plain.key))
        .toMatchObject({ hidden: false, calculation: c.key });
    });

    it("should answer hidden for an output whose calculation the reader cannot retrieve", async () => {
      const c = await client.calculations.create({
        name: `calc_output_${id.create()}`,
        expression: "return 2",
      });
      const reader = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [channel.resourceID(c.output)],
        actions: ["retrieve"],
      });
      expect(await reader.calculations.outputs.retrieve([c.output])).toEqual([
        { key: c.output, hidden: true },
      ]);
    });

    it("should map an output to none when its calculation is deleted", async () => {
      const c = await client.calculations.create({
        name: `calc_output_${id.create()}`,
        expression: "return 2",
      });
      const local = createTestClient();
      await local.connect();
      await local.calculations.outputs.retrieve([c.output]);
      await client.calculations.delete(c.key);
      await expect
        .poll(() => local.calculations.outputs.get(c.output))
        .toMatchObject({ hidden: false, calculation: null });
    });
  });

  describe("isOutput", () => {
    const key = "2a7d9c1e-5b3f-4e60-9a8b-0c1d2e3f4a5b";
    it.each<[string, calculation.Output, boolean]>([
      ["a hidden output", { key: 1, hidden: true }, true],
      ["an output", { key: 1, hidden: false, calculation: key }, true],
      [
        "a channel that no calculation writes",
        { key: 1, hidden: false, calculation: null },
        false,
      ],
    ])("should answer for %s", (_, output, expected) => {
      expect(calculation.isOutput(output)).toBe(expected);
    });
  });

  describe("status", () => {
    const remote = createTestClient();

    const setStatus = async (key: calculation.Key, message: string): Promise<void> => {
      await remote.statuses.set({
        owner: calculation.resourceID(key),
        name: "Calculation Status",
        variant: "info",
        message,
        time: TimeStamp.now(),
      });
    };

    describe("of an output", () => {
      it("should retrieve the status of the calculation that writes a channel", async () => {
        const c = await client.calculations.create({
          name: `calc_status_${id.create()}`,
          expression: "return 2",
        });
        await setStatus(c.key, "calculating");
        const local = createTestClient();
        expect(local.calculations.getCachedStatus(c.output)).toBeUndefined();
        expect(await local.calculations.retrieveStatus(c.output)).toMatchObject({
          message: "calculating",
        });
        expect(local.calculations.getCachedStatus(c.output)).toMatchObject({
          message: "calculating",
        });
      });

      it("should retrieve null for a channel that no calculation writes", async () => {
        const plain = await createVirtual();
        const local = createTestClient();
        expect(await local.calculations.retrieveStatus(plain.key)).toBeNull();
        expect(local.calculations.getCachedStatus(plain.key)).toBeNull();
      });

      it("should retrieve null for a calculation the reader cannot retrieve", async () => {
        const c = await client.calculations.create({
          name: `calc_status_${id.create()}`,
          expression: "return 2",
        });
        await setStatus(c.key, "calculating");
        const reader = await createTestClientWithPolicy(client, {
          name: id.create(),
          objects: [channel.resourceID(c.output)],
          actions: ["retrieve"],
        });
        expect(await reader.calculations.retrieveStatus(c.output)).toBeNull();
        expect(reader.calculations.getCachedStatus(c.output)).toBeNull();
      });
    });

    it("should compose the status only for a request that includes it", async () => {
      const c = await client.calculations.create({
        name: `calc_status_${id.create()}`,
        expression: "return 2",
      });
      await setStatus(c.key, "calculating");
      const local = createTestClient();
      await local.connect();
      // The status is cached first, so the bare retrieve shows the request gating it.
      const [withStatus] = await local.calculations.retrieve({
        output: c.output,
        includeStatus: true,
      });
      expect(withStatus.status?.message).toEqual("calculating");
      const [bare] = await local.calculations.retrieve({ output: c.output });
      expect(bare.status).toBeUndefined();
    });

    it("should deliver a remote status change to a request that includes it", async () => {
      const c = await client.calculations.create({
        name: `calc_status_${id.create()}`,
        expression: "return 2",
      });
      await setStatus(c.key, "calculating");
      const list = client.calculations.watch({ output: c.output, includeStatus: true });
      const results: Array<calculation.Calculation[] | undefined> = [];
      const off = list.onChange((w) => results.push(w?.records));
      try {
        await list.retrieve();
        await setStatus(c.key, "done");
        await expect.poll(() => results.at(-1)?.[0]?.status?.message).toEqual("done");
      } finally {
        off();
      }
    });
  });

  describe("dispatch", () => {
    it("should set the expression of a calculation", async () => {
      const c = await client.calculations.create({
        name: `calc_output_${id.create()}`,
        expression: "return 2",
      });
      expect(
        await client.calculations.dispatch(
          c.key,
          calculation.set({ path: ["expression"], value: "return 3" }),
        ),
      ).toBe(true);
      await expect
        .poll(
          async () => (await client.calculations.retrieve({ key: c.key })).expression,
        )
        .toEqual("return 3");
    });
  });

  describe("delete", () => {
    it("should delete a calculation and keep its output", async () => {
      const c = await client.calculations.create({
        name: `calc_output_${id.create()}`,
        expression: "return 2",
      });
      await client.calculations.delete(c.key);
      await expect(client.calculations.retrieve({ output: c.output })).resolves.toEqual(
        [],
      );
      await expect(client.channels.retrieve(c.output)).resolves.toBeDefined();
    });
  });
});
