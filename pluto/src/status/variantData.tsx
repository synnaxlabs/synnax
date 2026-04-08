// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/x";

import { type Select } from "@/select";
import { Indicator } from "@/status/base/Indicator";

export const VARIANT_DATA: Select.StaticEntry<status.Variant>[] = [
  { key: "success", name: "Success", icon: <Indicator variant="success" /> },
  { key: "error", name: "Error", icon: <Indicator variant="error" /> },
  { key: "warning", name: "Warning", icon: <Indicator variant="warning" /> },
  { key: "info", name: "Info", icon: <Indicator variant="info" /> },
  { key: "loading", name: "Loading", icon: <Indicator variant="loading" /> },
  { key: "disabled", name: "Disabled", icon: <Indicator variant="disabled" /> },
];
