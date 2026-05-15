// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/x/status";
import { type Select } from "@synnaxlabs/charon/select";
import { Status } from "@synnaxlabs/charon/status";

export const VARIANT_DATA: Select.StaticEntry<status.Variant>[] = [
  { key: "success", name: "Success", icon: <Status.Indicator variant="success" /> },
  { key: "error", name: "Error", icon: <Status.Indicator variant="error" /> },
  { key: "warning", name: "Warning", icon: <Status.Indicator variant="warning" /> },
  { key: "info", name: "Info", icon: <Status.Indicator variant="info" /> },
  { key: "loading", name: "Loading", icon: <Status.Indicator variant="loading" /> },
  { key: "disabled", name: "Disabled", icon: <Status.Indicator variant="disabled" /> },
];
