// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/x";

export const VARIANT_COLORS: Record<status.Variant, string> = {
  info: "var(--pluto-text-color)",
  error: "var(--pluto-error-z)",
  warning: "var(--pluto-warning-m1)",
  success: "var(--pluto-primary-z)",
  loading: "var(--pluto-text-color)",
  disabled: "var(--pluto-gray-l8)",
};
