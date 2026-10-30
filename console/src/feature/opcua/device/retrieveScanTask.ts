// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status, type Synnax, type task } from "@synnaxlabs/client";
import { type record, TimeSpan } from "@synnaxlabs/x";
import { type z } from "zod";

import { type SCAN_SCHEMAS, SCAN_TYPE } from "@/feature/opcua/task/types";

export const retrieveScanTask = async (
  client: Synnax,
  rack: number,
): Promise<task.Task<typeof SCAN_SCHEMAS>> =>
  await client.tasks.retrieve<typeof SCAN_SCHEMAS>({ type: SCAN_TYPE, rack });

interface ExecuteScanCommandParams<Z extends z.ZodType> {
  type: string;
  args: record.Unknown;
  /** The schema of the data of the reply. */
  schema: Z;
  timeout?: TimeSpan;
}

/**
 * Runs a command on the OPC UA scan task of a rack and waits for its reply.
 * @returns The data of the reply.
 * @throws {Error} if the command fails, with the status of the failure, or if the data
 * of the reply does not match the schema.
 */
export const executeScanCommand = async <Z extends z.ZodType>(
  client: Synnax,
  rack: number,
  { type, args, schema, timeout = TimeSpan.seconds(10) }: ExecuteScanCommandParams<Z>,
): Promise<z.infer<Z>> => {
  const scanTask = await retrieveScanTask(client, rack);
  const res = await scanTask.executeCommandSync({ type, timeout, args });
  if (res.variant === "error") throw status.toError(res);
  return schema.parse(res.details?.data);
};
