// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { record } from "@synnaxlabs/x";
import { z } from "zod";

export const SCAN_TYPE = "opc_scan";

const scannedNodeZ = z
  .object({
    key: z.string().optional(),
    dataType: z.string(),
    isArray: z.boolean(),
    name: z.string(),
    nodeClass: z.string(),
    nodeId: z.string(),
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

// Each command replies with its own shape, and the client drops a reply that fails
// this schema, so the union must cover every one.
const scanCommandResponseZ = z
  .union([browseResponseZ, certificatesResponseZ, testConnectionResponseZ])
  .nullish()
  .optional();

export const TEST_CONNECTION_COMMAND_TYPE = "test_connection";

export const BROWSE_COMMAND_TYPE = "browse";

export const GET_CERTIFICATE_COMMAND_TYPE = "get_certificate";

export const REGENERATE_CERTIFICATE_COMMAND_TYPE = "regenerate_certificate";

export const SCAN_SCHEMAS = {
  type: z.literal(SCAN_TYPE),
  config: record.nullishToEmpty(),
  statusData: scanCommandResponseZ,
} as const satisfies task.Schemas;
