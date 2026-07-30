// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export {
  check,
  type CheckParams,
  Client,
  type ClientParams,
  type FactEvent,
} from "@/connection/client";
export {
  type Config,
  DEFAULT_STATUS,
  type Handle,
  type Info,
  type Reason,
  REASONS,
  reasonZ,
  type Status,
  type StatusDetails,
  statusDetailsZ,
  statusZ,
} from "@/connection/status";
