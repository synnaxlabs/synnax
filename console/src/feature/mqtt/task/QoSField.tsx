// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component, Form as PForm, Select } from "@synnaxlabs/pluto";
import { type FC } from "react";

import { type QoS } from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";

const DATA: Select.StaticEntry<QoS>[] = [
  { key: "at_most_once", name: "At most once (0)" },
  { key: "at_least_once", name: "At least once (1)" },
  { key: "exactly_once", name: "Exactly once (2)" },
];

const renderSelect = Component.renderProp(
  (
    p: Omit<Select.StaticProps<QoS, Select.StaticEntry<QoS>>, "data" | "resourceName">,
  ) => (
    <Select.Static<QoS, Select.StaticEntry<QoS>>
      {...p}
      data={DATA}
      resourceName="quality of service"
    />
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
