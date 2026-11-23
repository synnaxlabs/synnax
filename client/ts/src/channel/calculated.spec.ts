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

const SUBJECTS: Record<string, () => Promise<Synnax>> = {
  "an admin that never retrieved the calculation": async () => createTestClient(),
  "a reader that cannot read the calculation": async () =>
    await createTestClientWithPolicy(admin, {
      name: id.create(),
      objects: [channel.resourceID(0), framer.TYPE_RESOURCE_ID],
      actions: ["retrieve"],
    }),
};

const WHERES: Record<string, (prefix: string) => channel.Where> = {
  calculated: (prefix) => ({ calculated: true, name: { startsWith: prefix } }),
  "not virtual or calculated": (prefix) => ({
    or: [{ virtual: false }, { calculated: true }],
    name: { startsWith: prefix },
  }),
};

describe("live calculated lists", () => {
  Object.entries(SUBJECTS).forEach(([subject, createSubject]) =>
    Object.entries(WHERES).forEach(([line, whereOf]) =>
      it(`should admit and drop an output for ${subject}, ${line}`, async () => {
        const prefix = `calculated_${id.create()}`;
        const output = await admin.channels.create({
          name: `${prefix}_output`,
          dataType: DataType.FLOAT32,
          virtual: true,
        });
        const client = await createSubject();
        await client.connect();
        await waitForStreamLive(client.connection);
        const list = client.channels.watch({ where: whereOf(prefix) });
        const off = list.onChange(() => {});
        try {
          await list.retrieve();
          const keys = () => (list.getCached()?.records ?? []).map(({ key }) => key);
          expect(keys()).toEqual([]);
          const calc = await admin.calculations.create({
            output: output.key,
            expression: "return 2",
          });
          await expect.poll(keys).toEqual([output.key]);
          await admin.calculations.delete(calc.key);
          await expect.poll(keys).toEqual([]);
        } finally {
          off();
        }
      }),
    ),
  );
});
