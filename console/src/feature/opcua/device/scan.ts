// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { task } from "@synnaxlabs/client";
import { record } from "@synnaxlabs/x";
import { z } from "zod";

/**
 * @returns a key for what a channel reads or writes: its node, and the member of the
 * node's structure value when it has a field.
 */
export const nodeIdentity = (ch: { nodeId: string; field?: string }) =>
  JSON.stringify([ch.nodeId, ch.field ?? ""]);

export const SCAN_TYPE = "opc_scan";

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

export interface ScannedNode extends z.infer<typeof scannedNodeZ> {}

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
  clientCertificateRejected: z.boolean().optional(),
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
