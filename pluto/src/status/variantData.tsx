// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Select } from "@synnaxlabs/lyra/select";
import { Status as Base } from "@synnaxlabs/lyra/status";

/** The status variants as selectable entries, each with its name and icon. */
export const VARIANT_DATA: Select.StaticEntry<Base.Variant>[] = [
  { key: "success", name: "Success", icon: <Base.Indicator variant="success" /> },
  { key: "error", name: "Error", icon: <Base.Indicator variant="error" /> },
  { key: "warning", name: "Warning", icon: <Base.Indicator variant="warning" /> },
  { key: "info", name: "Info", icon: <Base.Indicator variant="info" /> },
  { key: "loading", name: "Loading", icon: <Base.Indicator variant="loading" /> },
  { key: "disabled", name: "Disabled", icon: <Base.Indicator variant="disabled" /> },
];
