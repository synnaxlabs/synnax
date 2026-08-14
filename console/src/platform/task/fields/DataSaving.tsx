// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form, Input } from "@synnaxlabs/pluto";

import { type Polarity } from "@/platform/task/types";

export interface DataSavingProps {
  polarity?: Polarity;
}

export const DataSaving = ({ polarity = "enabled" }: DataSavingProps) => {
  if (polarity === "enabled")
    return <Form.SwitchField label="Data saving" path="config.dataSaving" />;
  return (
    <Form.Field<boolean> label="Data saving" path="config.dataSavingDisabled">
      {({ value, onChange, ...rest }) => (
        <Input.Switch {...rest} value={!value} onChange={(v) => onChange(!v)} />
      )}
    </Form.Field>
  );
};
