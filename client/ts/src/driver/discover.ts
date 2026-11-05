// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { status } from "@/status";

/** The outcome of one stage of a connection test. */
export const testStepZ = z.object({
  /** stage names the stage, such as "dial" or "tls". */
  stage: z.string(),
  /** address is the network address that the stage reached, or empty. */
  address: z.string(),
  /** variant is success, warning, or error. */
  variant: status.variantZ,
  /** message says what happened in one short line. */
  message: z.string(),
  /** description says why, and what to do now. */
  description: z.string(),
});

export interface TestStep extends z.infer<typeof testStepZ> {}

/** The data that answers a connection test command of a scan task. */
export const testResultZ = z.object({
  /** steps are the stages that ran, in order. The last is the first that failed. */
  steps: z.array(testStepZ),
});

export interface TestResult extends z.infer<typeof testResultZ> {}

/** The encodings of a message payload. */
export const FORMATS = ["json", "schema_registry", "text", "binary"] as const;

export const formatZ = z.enum(FORMATS);

export type Format = z.infer<typeof formatZ>;

/** Decodes the base64 form of a message payload into its bytes. */
export const payloadZ = z
  .string()
  .transform((encoded) => Uint8Array.from(atob(encoded), (c) => c.charCodeAt(0)));
