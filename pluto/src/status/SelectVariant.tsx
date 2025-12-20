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
import { Indicator } from "@/status/core/Indicator";

const DATA: Core.StaticEntry<status.Variant>[] = [
  {
    key: "success",
    name: "Success",
    icon: <Indicator variant="success" />,
  },
  {
    key: "error",
    name: "Error",
    icon: <Indicator variant="error" />,
  },
  {
    key: "warning",
    name: "Warning",
    icon: <Indicator variant="warning" />,
  },
  {
    key: "info",
    name: "Info",
    icon: <Indicator variant="info" />,
  },
];

export interface SelectVariantProps extends Omit<
  Core.StaticProps<status.Variant>,
  "data" | "resourceName"
> {}

export const SelectVariant = (props: SelectVariantProps): ReactElement => (
  <Core.Static {...props} data={DATA} resourceName="status variant" />
);
