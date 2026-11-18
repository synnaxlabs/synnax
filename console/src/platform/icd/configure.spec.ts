// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { ICD } from "@/platform/icd";
import { uniqueName } from "@/testutil";

const client = createTestClient();

describe("configureMessage", () => {
  it("should give a field whose channel was deleted a new channel", async () => {
    const prefix = uniqueName("src");
    const index = await client.channels.create({
      name: `${prefix}_old_time`,
      dataType: "timestamp",
      isIndex: true,
    });
    const old = await client.channels.create({
      name: `${prefix}_old`,
      dataType: "float64",
      index: index.key,
    });
    await client.channels.delete(old.key);
    const field = icd.jsonFieldZ.parse({
      key: uuid.create(),
      name: "temp",
      pointer: "/temp",
      type: "float",
    });
    const message: icd.Message = {
      key: uuid.create(),
      name: "Status",
      payload: { format: "json", fields: [field] },
    };
    const bound = {
      source: {
        icd: uuid.create(),
        disabled: false,
        messages: [
          {
            message: message.key,
            bindings: [{ field: field.key, channel: old.key, dataType: "" }],
          },
        ],
      },
      read: 0,
      message,
      fields: [field],
    };
    const [key] = await ICD.configureMessage(client, bound, prefix);
    expect(key).not.toBe(old.key);
    expect((await client.channels.retrieve(key)).name).toBe(`${prefix}_temp`);
    expect(bound.source.messages[0].bindings[0].channel).toBe(key);
  });
});
