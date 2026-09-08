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

import { type channel } from "@/channel";
import { type RetrieveMultipleParams } from "@/control/client";
import { type KeyedState } from "@/control/state";
import { NotFoundError } from "@/errors";
import { query } from "@/query";
import { createTestClient } from "@/testutil";

const client = createTestClient();

const createVirtual = async () =>
  await client.channels.create({
    name: `control_${id.create()}`,
    dataType: DataType.FLOAT64,
    virtual: true,
  });

const keysOf = (states: query.Cached<KeyedState[]> | undefined): channel.Key[] =>
  query.isLive<KeyedState[]>(states) ? states.map(({ key }) => key) : [];

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

  describe("onChange", () => {
    it("should deliver a live transfer when no keys are given", async () => {
      const ch = await createVirtual();
      const all: RetrieveMultipleParams = {};
      const off = client.control.onChange(all, () => {});
      try {
        await client.control.retrieve(all);
        const w = await client.openWriter({
          start: TimeStamp.now(),
          channels: [ch.key],
          controlSubject: { key: "helena", name: "helena" },
        });
        try {
          await expect
            .poll(() => keysOf(client.control.getCached(all)))
            .toContain(ch.key);
        } finally {
          await w.close();
        }
      } finally {
        off();
      }
    });

    it("should not deliver a live transfer for an empty key set", async () => {
      const ch = await createVirtual();
      const none: channel.Key[] = [];
      const all: RetrieveMultipleParams = {};
      const delivered: channel.Key[][] = [];
      const offNone = client.control.onChange(none, (states) =>
        delivered.push(keysOf(states)),
      );
      const offAll = client.control.onChange(all, () => {});
      try {
        expect(await client.control.retrieve(none)).toHaveLength(0);
        await client.control.retrieve(all);
        const w = await client.openWriter({
          start: TimeStamp.now(),
          channels: [ch.key],
          controlSubject: { key: "missoula", name: "missoula" },
        });
        try {
          // The unfiltered query witnesses the transfer landing in the table, which
          // rechecks every subscribed query in the same batch.
          await expect
            .poll(() => keysOf(client.control.getCached(all)))
            .toContain(ch.key);
          expect(keysOf(client.control.getCached(none))).toEqual([]);
          expect(delivered.flat()).toEqual([]);
        } finally {
          await w.close();
        }
      } finally {
        offNone();
        offAll();
      }
    });
  });
});
