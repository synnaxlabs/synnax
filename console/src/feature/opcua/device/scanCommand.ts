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

interface SendScanCommandParams {
  type: string;
  args: record.Unknown;
  timeout?: TimeSpan;
}

/**
 * Runs a command on the OPC UA scan task of a rack and waits for its reply.
 * @returns The status of the reply, which can be a failure.
 */
export const sendScanCommand = async (
  client: Synnax,
  rack: number,
  { type, args, timeout = TimeSpan.seconds(10) }: SendScanCommandParams,
): Promise<task.Status<(typeof SCAN_SCHEMAS)["statusData"]>> => {
  const scanTask = await client.tasks.retrieve<typeof SCAN_SCHEMAS>({
    type: SCAN_TYPE,
    rack,
  });
  return await scanTask.executeCommandSync({ type, timeout, args });
};

interface ExecuteScanCommandParams<Z extends z.ZodType> extends SendScanCommandParams {
  /** The schema of the data of the reply. */
  schema: Z;
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
  { schema, ...params }: ExecuteScanCommandParams<Z>,
): Promise<z.infer<Z>> => {
  const res = await sendScanCommand(client, rack, params);
  if (res.variant === "error") throw status.toError(res);
  return schema.parse(res.details?.data);
};
