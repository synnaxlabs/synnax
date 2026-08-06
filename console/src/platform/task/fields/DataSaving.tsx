// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form, Input } from "@synnaxlabs/pluto";

export const DataSaving = () => (
    <Form.Field<boolean> label="Data saving" path="config.dataSavingDisabled">
      {({ value, onChange, ...rest }) => (
        <Input.Switch {...rest} value={!value} onChange={(v) => onChange(!v)} />
      )}
    </Form.Field>
);
