// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@/status";

export const LOADING: status.Crude = {
  variant: "loading",
  message: "Loading channel data",
};

export const SUCCESS: status.Crude = {
  variant: "success",
  message: "Loaded channel data",
};

export const READ_FAILED = "Failed to read channel data";

export const NO_HISTORY: status.Crude = {
  variant: "warning",
  message: "Failed to read calculated channel history",
  description: "The channel shows only the samples streamed since it opened.",
};
