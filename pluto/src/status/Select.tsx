// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Select as Core } from "@/select";
import { Status } from "@/status";

const DATA: Core.SimplyEntry<status.Variant>[] = [
  {
    key: "success",
    name: "Success",
    icon: <Status.Indicator variant="success" />,
  },
  {
    key: "error",
    name: "Error",
    icon: <Status.Indicator variant="error" />,
  },
  {
    key: "warning",
    name: "Warning",
    icon: <Status.Indicator variant="warning" />,
  },
  {
    key: "info",
    name: "Info",
    icon: <Status.Indicator variant="info" />,
  },
];

export interface SelectProps
  extends Omit<Core.SimpleProps<status.Variant>, "data" | "resourceName"> {}

export const Select = (props: SelectProps): ReactElement => (
  <Core.Simple {...props} data={DATA} resourceName="Status" />
);
