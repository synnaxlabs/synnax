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
export type WriteChannel = opcua.WriteChannel;
export type Channel = ReadChannel | WriteChannel;
export type MethodArgument = opcua.MethodArgument;
export interface EventField extends opcua.EventField {}

/** @returns true if the channel calls a method instead of writing a node. */
export const isMethod = (ch: Channel): ch is opcua.MethodWriteChannel =>
  "type" in ch && ch.type === "method";

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
  // A method can take more than one channel, each with its own arguments.
  channels.forEach((ch) => {
    if (isMethod(ch)) return;
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
    const { arrayMode, arraySize, channels, mode, sampleRate, streamRate } = value;
    if (mode === "on_change")
      channels.forEach(({ isIndex }, i) => {
        if (isIndex)
          issues.push({
            code: "custom",
            message: "On change mode cannot read an index channel",
            path: ["channels", i, "isIndex"],
            input: value,
          });
      });
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

const validateMethodChannels = ({
  value: channels,
  issues,
}: z.core.ParsePayload<WriteChannel[]>) =>
  channels.forEach((ch, i) => {
    if (ch.type !== "method" || ch.disabled) return;
    if (ch.objectId.length === 0)
      issues.push({
        code: "custom",
        message: "Method must have an object",
        path: [i, "objectId"],
        input: channels,
      });
    const commands = ch.arguments.filter(({ source }) => source === "command");
    if (commands.length > 1)
      issues.push({
        code: "custom",
        message: "Only one argument can take the command",
        path: [i, "arguments"],
        input: channels,
      });
  });

export const deployWriteConfigZ = opcua.writeConfigZ.extend({
  device: Task.deviceKeyZ,
  channels: opcua.writeChannelZ
    .array()
    .check(Task.validateChannels)
    .check(Task.validateEnabledChannel)
    .check(validateMethodChannels)
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

export const EVENT_TYPE = `${PREFIX}_event`;

// The keys of the fields that every event records.
const STANDARD_EVENT_KEYS = [
  "event_id",
  "event_type",
  "source_name",
  "time",
  "message",
  "severity",
];

const NAMESPACE_URI_PREFIX = "nsu=";

// Returns the problem with a browse path, by the rules that the Driver parses it with.
const findBrowsePathIssue = (path: string): string | null => {
  let rest = path;
  let more = true;
  while (more) {
    let named = false;
    if (
      rest.slice(0, NAMESPACE_URI_PREFIX.length).toLowerCase() === NAMESPACE_URI_PREFIX
    ) {
      const end = rest.indexOf(";");
      if (end === -1) return "Browse path must end each namespace URI with ';'";
      const uri = rest.slice(NAMESPACE_URI_PREFIX.length, end);
      if (uri.length === 0 || /%(?![0-9a-f]{2})/i.test(uri))
        return "Browse path has an invalid namespace URI";
      rest = rest.slice(end + 1);
      named = true;
    }
    const slash = rest.indexOf("/");
    let name = slash === -1 ? rest : rest.slice(0, slash);
    const index = /^(\d+):/.exec(name);
    if (!named && index != null && Number(index[1]) <= 0xffff)
      name = name.slice(index[0].length);
    if (name.length === 0) return "Browse path must name each element";
    more = slash !== -1;
    rest = rest.slice(slash + 1);
  }
  return null;
};

export const deployEventConfigZ = opcua.eventConfigZ.extend({
  device: Task.deviceKeyZ,
  fields: opcua.eventFieldZ.array().check(({ value: fields, issues }) => {
    const names = new Set<string>();
    fields.forEach(({ name, browsePath }, i) => {
      if (name.length === 0)
        issues.push({
          code: "custom",
          message: "Field must have a name",
          path: [i, "name"],
          input: fields,
        });
      else if (STANDARD_EVENT_KEYS.includes(name))
        issues.push({
          code: "custom",
          message: "A standard field has this name",
          path: [i, "name"],
          input: fields,
        });
      else if (names.has(name))
        issues.push({
          code: "custom",
          message: "Another field has this name",
          path: [i, "name"],
          input: fields,
        });
      names.add(name);
      const pathIssue = findBrowsePathIssue(browsePath);
      if (pathIssue != null)
        issues.push({
          code: "custom",
          message: pathIssue,
          path: [i, "browsePath"],
          input: fields,
        });
    });
  }),
});

export const EVENT_SCHEMAS = {
  type: z.literal(EVENT_TYPE),
  config: opcua.eventConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type EventSchemas = typeof EVENT_SCHEMAS;

export const SCAN_TYPE = `${PREFIX}_scan`;

const scannedNodeFieldsZ = z.object({
  key: z.string().optional(),
  dataType: z.string(),
  isArray: z.boolean(),
  name: z.string(),
  nodeClass: z.string(),
  nodeId: z.string(),
  field: z.string().default(""),
});

const withKey = <N extends { key?: string; nodeId: string }>({ key, ...rest }: N) => ({
  ...rest,
  key: key ?? rest.nodeId,
});

const scannedNodeZ = scannedNodeFieldsZ.transform(withKey);

const inputArgumentZ = z.object({
  name: z.string(),
  typeName: z.string(),
  dataType: z.string(),
});

/** An input argument of a method, as the server declares it. */
export interface InputArgument extends z.infer<typeof inputArgumentZ> {}

export interface ScannedNode extends z.infer<typeof scannedNodeZ> {
  /** The node ID of the node that the browser found this one under. */
  parent?: string;
}

// A variable that a search under a folder found, with its browse path below it.
const foundNodeZ = scannedNodeFieldsZ
  .extend({ path: z.array(z.string()) })
  .transform(withKey);

export const browseVariablesResponseZ = z.object({
  channels: z.array(foundNodeZ),
  truncated: z.boolean(),
  /** Why each object under the folder that failed to browse was skipped. */
  skipped: z.array(z.string()),
});

export const nodeDetailsZ = z.object({
  node: scannedNodeZ,
  nodeClass: z.string(),
  browseName: z.string(),
  displayName: z.string().optional(),
  description: z.string().optional(),
  typeName: z.string().optional(),
  valueRank: z.number().optional(),
  arrayDimensions: z.array(z.number()).optional(),
  readable: z.boolean().optional(),
  writable: z.boolean().optional(),
  historyReadable: z.boolean().optional(),
  historizing: z.boolean().optional(),
  minimumSamplingInterval: z.number().optional(),
  status: z.string().optional(),
  sourceTimestamp: z.string().optional(),
  serverTimestamp: z.string().optional(),
  value: z.string().optional(),
  inputArguments: z.array(inputArgumentZ).optional(),
});

/** The attributes and the value of one node, as the server reports them. */
export interface NodeDetails extends z.infer<typeof nodeDetailsZ> {}

const certificateInfoZ = z.object({
  pem: z.string(),
  thumbprint: z.string(),
  subject: z.string(),
  notAfter: z.string(),
});

export interface CertificateInfo extends z.infer<typeof certificateInfoZ> {}

export const browseResponseZ = z.object({
  channels: z.array(scannedNodeZ),
  /** Whether the Driver stopped at a limit before it returned every child. */
  truncated: z.boolean().default(false),
});

export interface BrowseResponse extends z.infer<typeof browseResponseZ> {}

export const certificatesResponseZ = z.object({
  current: certificateInfoZ,
  next: certificateInfoZ.optional(),
});

export interface CertificatesResponse extends z.infer<typeof certificatesResponseZ> {}

export const testConnectionResponseZ = z.object({
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

export const TEST_CONNECTION_COMMAND_TYPE = "test_connection";

export const BROWSE_COMMAND_TYPE = "browse";

export const BROWSE_VARIABLES_COMMAND_TYPE = "browse_variables";

export const READ_NODE_COMMAND_TYPE = "read_node";

export const GET_ENDPOINTS_COMMAND_TYPE = "get_endpoints";

export const GET_CERTIFICATE_COMMAND_TYPE = "get_certificate";

export const REGENERATE_CERTIFICATE_COMMAND_TYPE = "regenerate_certificate";

export const SCAN_SCHEMAS = {
  type: z.literal(SCAN_TYPE),
  config: record.nullishToEmpty(),
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;
