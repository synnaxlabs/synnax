// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { channel } from "@/channel";
import type Synnax from "@/client";
import { framer } from "@/framer";
import {
  createTestClient,
  createTestClientWithPolicy,
  waitForStreamLive,
} from "@/testutil";

const admin = createTestClient();

const createVirtual = async (name: string): Promise<channel.Channel> =>
  await admin.channels.create({ name, dataType: DataType.FLOAT32, virtual: true });

const keysOf = (records: channel.Channel[] | undefined): channel.Key[] =>
  (records ?? []).map(({ key }) => key).sort();

// Data channels only, so a calculation's index stays out.
const whereOf = (virtual: boolean, prefix: string) => ({
  virtual,
  isIndex: { eq: false },
  name: { startsWith: prefix },
});

const clientOf = async (subject: "admin" | "reader"): Promise<Synnax> => {
  const c =
    subject === "admin"
      ? createTestClient()
      : await createTestClientWithPolicy(admin, {
          name: id.create(),
          objects: [channel.resourceID(0), framer.TYPE_RESOURCE_ID],
          actions: ["retrieve"],
        });
  await c.connect();
  await waitForStreamLive(c.connection);
  return c;
};

// What the Core answers now, through a client that holds no list.
const truthOf = async (subject: "admin" | "reader", virtual: boolean, prefix: string) =>
  keysOf(
    await (
      await clientOf(subject)
    ).channels.retrieve({ where: channel.whereZ.parse(whereOf(virtual, prefix)) }),
  );

// A live list, read as the Pluto list hooks read it: the cached window.
const watched = async (c: Synnax, virtual: boolean, prefix: string) => {
  const list = c.channels.watch({ where: whereOf(virtual, prefix) });
  const off = list.onChange(() => {});
  await list.retrieve();
  return { cached: () => keysOf(list.getCached()?.records), off };
};

describe("virtual line breaker", () => {
  for (const subject of ["admin", "reader"] as const)
    describe(subject, () => {
      it("should admit a virtual channel to a virtual: false list when a calculation starts to write it", async () => {
        const prefix = `vb_${id.create()}`;
        const plain = await createVirtual(`${prefix}_a`);
        const { cached, off } = await watched(await clientOf(subject), false, prefix);
        try {
          expect(cached()).toEqual([]);
          await admin.calculations.create({
            output: plain.key,
            expression: "return 2",
          });
          expect(await truthOf(subject, false, prefix)).toEqual([plain.key]);
          await expect.poll(cached, { timeout: 3000 }).toEqual([plain.key]);
        } finally {
          off();
        }
      });

      it("should drop a virtual channel from a virtual: true list when a calculation starts to write it", async () => {
        const prefix = `vb_${id.create()}`;
        const plain = await createVirtual(`${prefix}_a`);
        const { cached, off } = await watched(await clientOf(subject), true, prefix);
        try {
          expect(cached()).toEqual([plain.key]);
          await admin.calculations.create({
            output: plain.key,
            expression: "return 2",
          });
          expect(await truthOf(subject, true, prefix)).toEqual([]);
          await expect.poll(cached, { timeout: 3000 }).toEqual([]);
        } finally {
          off();
        }
      });

      // The expression keeps the data type, so the Core does not change the channel.
      const sameType = async () =>
        `return ${(await createVirtual(`in_${id.create()}`)).name} * 2`;

      it("should admit a virtual channel to a virtual: false list when a calculation of the same data type starts to write it", async () => {
        const prefix = `vb_${id.create()}`;
        const plain = await createVirtual(`${prefix}_a`);
        const expression = await sameType();
        const { cached, off } = await watched(await clientOf(subject), false, prefix);
        try {
          expect(cached()).toEqual([]);
          await admin.calculations.create({ output: plain.key, expression });
          expect(await truthOf(subject, false, prefix)).toEqual([plain.key]);
          await expect.poll(cached, { timeout: 3000 }).toEqual([plain.key]);
        } finally {
          off();
        }
      });

      it("should drop a virtual channel from a virtual: true list when a calculation of the same data type starts to write it", async () => {
        const prefix = `vb_${id.create()}`;
        const plain = await createVirtual(`${prefix}_a`);
        const expression = await sameType();
        const { cached, off } = await watched(await clientOf(subject), true, prefix);
        try {
          expect(cached()).toEqual([plain.key]);
          await admin.calculations.create({ output: plain.key, expression });
          expect(await truthOf(subject, true, prefix)).toEqual([]);
          await expect.poll(cached, { timeout: 3000 }).toEqual([]);
        } finally {
          off();
        }
      });

      it("should admit an output to a virtual: true list when its calculation is deleted", async () => {
        const prefix = `vb_${id.create()}`;
        const calc = await admin.calculations.create({
          name: `${prefix}_a`,
          expression: "return 2",
        });
        const { cached, off } = await watched(await clientOf(subject), true, prefix);
        try {
          expect(cached()).toEqual([]);
          await admin.calculations.delete(calc.key);
          expect(await truthOf(subject, true, prefix)).toEqual([calc.output]);
          await expect.poll(cached, { timeout: 3000 }).toEqual([calc.output]);
        } finally {
          off();
        }
      });

      it("should drop an output from a virtual: false list when its calculation is deleted", async () => {
        const prefix = `vb_${id.create()}`;
        const calc = await admin.calculations.create({
          name: `${prefix}_a`,
          expression: "return 2",
        });
        const { cached, off } = await watched(await clientOf(subject), false, prefix);
        try {
          expect(cached()).toEqual([calc.output]);
          await admin.calculations.delete(calc.key);
          expect(await truthOf(subject, false, prefix)).toEqual([]);
          await expect.poll(cached, { timeout: 3000 }).toEqual([]);
        } finally {
          off();
        }
      });

      it("should admit a new calculation's output to a virtual: false list", async () => {
        const prefix = `vb_${id.create()}`;
        const { cached, off } = await watched(await clientOf(subject), false, prefix);
        try {
          expect(cached()).toEqual([]);
          const calc = await admin.calculations.create({
            name: `${prefix}_a`,
            expression: "return 2",
          });
          await expect.poll(cached, { timeout: 3000 }).toEqual([calc.output]);
        } finally {
          off();
        }
      });

      it("should admit a new virtual channel to a virtual: true list", async () => {
        const prefix = `vb_${id.create()}`;
        const { cached, off } = await watched(await clientOf(subject), true, prefix);
        try {
          expect(cached()).toEqual([]);
          const plain = await createVirtual(`${prefix}_a`);
          await expect.poll(cached, { timeout: 3000 }).toEqual([plain.key]);
        } finally {
          off();
        }
      });

      it("should admit a channel renamed into a virtual: true list", async () => {
        const prefix = `vb_${id.create()}`;
        const plain = await createVirtual(`other_${id.create()}`);
        const { cached, off } = await watched(await clientOf(subject), true, prefix);
        try {
          expect(cached()).toEqual([]);
          await admin.channels.rename(plain.key, `${prefix}_renamed`);
          await expect.poll(cached, { timeout: 3000 }).toEqual([plain.key]);
        } finally {
          off();
        }
      });

      it("should drop a calculation's output renamed out of a virtual: false list", async () => {
        const prefix = `vb_${id.create()}`;
        const calc = await admin.calculations.create({
          name: `${prefix}_a`,
          expression: "return 2",
        });
        const { cached, off } = await watched(await clientOf(subject), false, prefix);
        try {
          expect(cached()).toEqual([calc.output]);
          await admin.channels.rename(calc.output, `other_${id.create()}`);
          await expect.poll(cached, { timeout: 3000 }).toEqual([]);
        } finally {
          off();
        }
      });

      it("should drop a deleted channel from a virtual: true list", async () => {
        const prefix = `vb_${id.create()}`;
        const plain = await createVirtual(`${prefix}_a`);
        const { cached, off } = await watched(await clientOf(subject), true, prefix);
        try {
          expect(cached()).toEqual([plain.key]);
          await admin.channels.delete(plain.key);
          await expect.poll(cached, { timeout: 3000 }).toEqual([]);
        } finally {
          off();
        }
      });
    });
});
