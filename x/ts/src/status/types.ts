// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const SUCCESS_VARIANT = "success";
export const INFO_VARIANT = "info";
export const WARNING_VARIANT = "warning";
export const ERROR_VARIANT = "error";

export const variantZ = z.enum([
  SUCCESS_VARIANT,
  INFO_VARIANT,
  WARNING_VARIANT,
  ERROR_VARIANT,
]);
export type Variant = z.infer<typeof variantZ>;
