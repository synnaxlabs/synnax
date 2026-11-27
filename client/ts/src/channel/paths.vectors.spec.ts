// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { readFileSync } from "node:fs";

import { DataType, id } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it } from "vitest";
import { z } from "zod";

import { group } from "@/group";
import { createTestClient } from "@/testutil";

const remote = createTestClient();

// The vectors pin the path that this client builds to the path that the Core builds.
const VECTORS_URL = new URL(
  "../../../../core/testdata/paths/vectors.json",
  import.meta.url,
);

const vectorsZ = z.object({
  cases: z
    .object({
      name: z.string(),
      folders: z.string().array(),
      channel: z.string(),
      path: z.string(),
    })
    .array(),
});

const { cases } = vectorsZ.parse(JSON.parse(readFileSync(VECTORS_URL, "utf-8")));

describe("channel paths shared with the Core", () => {
  beforeAll(async () => await remote.connect());

  it.each(cases)(
    "should build the path of $name as the Core builds it",
    async ({ folders, channel: name, path }) => {
      // A case with no folders makes its channel at the top.
      const top =
        folders.length === 0
          ? undefined
          : await remote.groups.create({ name: id.create(), scope: "channel" });
      let parent = top;
      for (const folder of folders)
        parent = await remote.groups.create({
          name: folder,
          parent: group.resourceID((parent as group.Group).key),
        });
      const suffix = `_${id.create()}`;
      const ch = await remote.channels.create({
        name: `${name}${suffix}`,
        dataType: DataType.FLOAT32,
        virtual: true,
        group: parent?.key,
      });
      // A client that the change stream told of no folder must fetch each one.
      const later = createTestClient();
      await later.connect();
      const res = await later.channels.retrieve({ key: ch.key, includePath: true });
      const prefix = top == null ? "" : `${top.name}.`;
      expect(res.path).toEqual(`${prefix}${path}${suffix}`);
    },
  );
});
