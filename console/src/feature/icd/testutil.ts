// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd, mqtt, type Synnax as Client, type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";

export const client = createTestClient();

/** A CSV import that holds one message, Engine, with one field, Rpm. */
export const ENGINE_CSV = "message,field,start_bit,bit_length\nEngine,Rpm,0,16";

/** Creates an ICD on the Core with a unique name and the given messages. */
export const createICD = async (
  messages: icd.Message[] = [],
  c: Client = client,
): Promise<icd.ICD> => await c.icds.create({ name: uniqueName("icd"), messages });

/**
 * Creates an ICD that holds a Command message and the other messages, and a write task
 * that sends Command.
 */
export const createUsedICD = async (
  others: icd.Message[] = [],
): Promise<{ ic: icd.ICD; used: task.Task }> => {
  const message = icd.messageZ.parse({
    key: uuid.create(),
    name: "Command",
    payload: { format: "json", fields: [] },
  });
  const ic = await createICD([message, ...others]);
  const target = mqtt.plainWriteTargetZ.parse({
    type: "plain",
    topic: "plant/valve/set",
    icd: ic.key,
    message: { message: message.key, bindings: [] },
  });
  const used = await client.tasks.create({
    name: uniqueName("task"),
    type: "mqtt_write",
    config: mqtt.writeConfigZ.parse({ targets: [target] }),
  });
  return { ic, used };
};
