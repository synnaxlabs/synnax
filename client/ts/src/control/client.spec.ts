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

import { NotFoundError } from "@/errors";
import { createTestClient } from "@/testutil";

const client = createTestClient();

const createVirtual = async () =>
  await client.channels.create({
    name: `control_${id.create()}`,
    dataType: DataType.FLOAT64,
    virtual: true,
  });

describe("control client", () => {
  describe("retrieve", () => {
    it("should return the state of a channel a writer holds", async () => {
      const ch = await createVirtual();
      const w = await client.openWriter({
        start: TimeStamp.now(),
        channels: [ch.key],
        controlSubject: { key: "philadelphia", name: "philadelphia" },
      });
      try {
        const state = await client.control.retrieve(ch.key);
        expect(state.key).toEqual(ch.key);
        expect(state.resource).toEqual(ch.key);
        expect(state.subject.name).toEqual("philadelphia");
      } finally {
        await w.close();
      }
    });

    it("should omit a channel that no subject controls", async () => {
      const ch = await createVirtual();
      expect(await client.control.retrieve([ch.key])).toHaveLength(0);
    });

    it("should throw when a single uncontrolled channel is requested", async () => {
      const ch = await createVirtual();
      await expect(client.control.retrieve(ch.key)).rejects.toThrow(NotFoundError);
    });

    it("should return every controlled channel when no keys are given", async () => {
      const ch = await createVirtual();
      const w = await client.openWriter({
        start: TimeStamp.now(),
        channels: [ch.key],
        controlSubject: { key: "denver", name: "denver" },
      });
      try {
        const states = await client.control.retrieve({});
        expect(states.some(({ resource }) => resource === ch.key)).toBe(true);
      } finally {
        await w.close();
      }
    });

    it("should return no states when an empty key set is given", async () => {
      const ch = await createVirtual();
      const w = await client.openWriter({
        start: TimeStamp.now(),
        channels: [ch.key],
        controlSubject: { key: "boulder", name: "boulder" },
      });
      try {
        expect(await client.control.retrieve([])).toHaveLength(0);
      } finally {
        await w.close();
      }
    });
  });
});
