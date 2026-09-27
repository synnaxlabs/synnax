// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/lyra/component";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { type FC } from "react";

import { type QoS } from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";

const renderSelect = Component.renderProp(
  (p: Omit<Select.SingleSimpleProps<QoS>, "children" | "resourceName">) => (
    <Select.Simple<QoS> {...p} resourceName="quality of service">
      <Select.Item itemKey="at_most_once">At most once (0)</Select.Item>
      <Select.Item itemKey="at_least_once">At least once (1)</Select.Item>
      <Select.Item itemKey="exactly_once">Exactly once (2)</Select.Item>
    </Select.Simple>
  ),
);

export interface QoSFieldProps {
  path: string;
}

export const QoSField: FC<QoSFieldProps> = ({ path }) => (
  <PForm.Field<QoS> path={path} label="Quality of service" className={CSS.B("qos")}>
    {renderSelect}
  </PForm.Field>
);
