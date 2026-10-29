// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, opcua, type task } from "@synnaxlabs/client";
import { record } from "@synnaxlabs/x";
import { z } from "zod";

import { Task } from "@/platform/task";

export const PREFIX = "opc";

export interface ReadChannel extends opcua.ReadChannel {}
export interface WriteChannel extends opcua.WriteChannel {}
export type Channel = ReadChannel | WriteChannel;

/**
 * @returns a key for what a channel reads or writes: its node, and the member of the
 * node's structure value when it has a field.
 */
export const nodeIdentity = (ch: Pick<ReadChannel, "nodeId"> & { field?: string }) =>
  JSON.stringify([ch.nodeId, ch.field ?? ""]);

const validateNodeIDs = ({
  value: channels,
  issues,
}: z.core.ParsePayload<Channel[]>) => {
  const counts = new Map<string, number>();
  channels.forEach((ch) => {
    const id = nodeIdentity(ch);
    counts.set(id, (counts.get(id) ?? 0) + 1);
  });
  channels.forEach((ch, i) => {
    if (ch.nodeId.length === 0 || (counts.get(nodeIdentity(ch)) ?? 0) < 2) return;
    issues.push({
      code: "custom",
      path: [i, "nodeId"],
      message:
        "field" in ch && ch.field !== ""
          ? "This field has already been used elsewhere in the configuration"
          : "This node ID has already been used elsewhere in the configuration",
      params: { variant: "warning" },
      input: channels,
    });
  });
};

export const READ_TYPE = `${PREFIX}_read`;

export interface ReadConfig extends opcua.ReadConfig {}

const readConfigZ = opcua.readConfigZ;

const validateIndexChannels = ({
  value: channels,
  issues,
}: z.core.ParsePayload<ReadChannel[]>) => {
  const indexChannelIndexes = channels
    .map(({ isIndex }, i) => (isIndex ? i : -1))
    .filter((i) => i !== -1);
  if (indexChannelIndexes.length <= 1) return;
  indexChannelIndexes.forEach((i) => {
    issues.push({
      code: "custom",
      message: "Only one channel can be marked as an index channel",
      path: [i, "isIndex"],
      input: channels,
    });
  });
};

export const deployReadConfigZ = opcua.readConfigZ
  .extend({
    device: Task.deviceKeyZ,
    sampleRate: z.number().positive().max(10000),
    channels: opcua.readChannelZ
      .array()
      .check(Task.validateReadChannels)
      .check(Task.validateEnabledChannel)
      .check(validateNodeIDs)
      .check(validateIndexChannels),
  })
  .check((ctx) => {
    const { value, issues } = ctx;
    const { arrayMode, arraySize, sampleRate, streamRate } = value;
    if (arrayMode) {
      if (!Number.isInteger(arraySize) || arraySize <= 0)
        issues.push({
          code: "custom",
          message: "Array size must be a positive integer",
          path: ["arraySize"],
          input: value,
        });
      else if (sampleRate < arraySize)
        issues.push({
          code: "custom",
          message: "Sample rate must be greater than or equal to the array size",
          path: ["sampleRate"],
          input: value,
        });
      return;
    }
    if (streamRate <= 0 || streamRate > 10000)
      issues.push({
        code: "custom",
        message: "Stream rate must be between 0 and 10000",
        path: ["streamRate"],
        input: value,
      });
    else Task.validateStreamRate(ctx);
  });

export const READ_SCHEMAS = {
  type: z.literal(READ_TYPE),
  config: readConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type ReadSchemas = typeof READ_SCHEMAS;

export const WRITE_TYPE = `${PREFIX}_write`;

const writeConfigZ = opcua.writeConfigZ;

export const deployWriteConfigZ = opcua.writeConfigZ.extend({
  device: Task.deviceKeyZ,
  channels: opcua.writeChannelZ
    .array()
    .check(Task.validateChannels)
    .check(Task.validateEnabledChannel)
    .check(({ value: channels, issues }) => {
      const channelsToIndexMap = new Map<channel.Key, number>();
      channels.forEach(({ cmdChannel }, i) => {
        if (cmdChannel === 0) return;
        if (!channelsToIndexMap.has(cmdChannel)) {
          channelsToIndexMap.set(cmdChannel, i);
          return;
        }
        const index = channelsToIndexMap.get(cmdChannel) as number;
        const code = "custom";
        const message = `Synnax channel with key ${cmdChannel} is used for multiple channels`;
        issues.push({ code, message, path: [index, "cmdChannel"], input: channels });
        issues.push({ code, message, path: [i, "cmdChannel"], input: channels });
      });
    })
    .check(validateNodeIDs),
});

export const WRITE_SCHEMAS = {
  type: z.literal(WRITE_TYPE),
  config: writeConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type WriteSchemas = typeof WRITE_SCHEMAS;

export const SCAN_TYPE = `${PREFIX}_scan`;

const scannedNodeZ = z
  .object({
    key: z.string().optional(),
    dataType: z.string(),
    isArray: z.boolean(),
    name: z.string(),
    nodeClass: z.string(),
    nodeId: z.string(),
    field: z.string().default(""),
  })
  .transform(({ key, ...rest }) => ({ ...rest, key: key ?? rest.nodeId }));

export interface ScannedNode extends z.infer<typeof scannedNodeZ> {}

const certificateInfoZ = z.object({
  pem: z.string(),
  thumbprint: z.string(),
  subject: z.string(),
  notAfter: z.string(),
});

export interface CertificateInfo extends z.infer<typeof certificateInfoZ> {}

const browseResponseZ = z.object({ channels: z.array(scannedNodeZ) });

export const certificatesResponseZ = z.object({
  current: certificateInfoZ,
  next: certificateInfoZ.optional(),
});

export interface CertificatesResponse extends z.infer<typeof certificatesResponseZ> {}

const testConnectionResponseZ = z.object({
  serverCertificate: certificateInfoZ.optional(),
});

const endpointZ = z.object({
  url: z.string(),
  securityMode: z.string(),
  securityPolicy: z.string(),
  userTokenTypes: z.array(z.string()),
  securityLevel: z.number(),
});

/** A way that a server accepts connections, as the server describes it. */
export interface Endpoint extends z.infer<typeof endpointZ> {}

export const endpointsResponseZ = z.object({ endpoints: z.array(endpointZ) });

// Each command replies with its own shape, and the client drops a reply that fails this
// schema, so the union must cover every one. The test connection reply matches any
// object, so it comes last.
const scanCommandResponseZ = z
  .union([
    browseResponseZ,
    certificatesResponseZ,
    endpointsResponseZ,
    testConnectionResponseZ,
  ])
  .nullish()
  .optional();

export const TEST_CONNECTION_COMMAND_TYPE = "test_connection";

export const BROWSE_COMMAND_TYPE = "browse";

export const GET_ENDPOINTS_COMMAND_TYPE = "get_endpoints";

export const GET_CERTIFICATE_COMMAND_TYPE = "get_certificate";

export const REGENERATE_CERTIFICATE_COMMAND_TYPE = "regenerate_certificate";

export const SCAN_SCHEMAS = {
  type: z.literal(SCAN_TYPE),
  config: record.nullishToEmpty(),
  statusData: scanCommandResponseZ,
} as const satisfies task.Schemas;
