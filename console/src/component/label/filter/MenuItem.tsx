// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label } from "@synnaxlabs/client";
import { type Dialog, Form, type Select } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { SelectMultiple } from "@/label/Select";

const TRIGGER_PROPS: Select.MultipleTriggerProps<string> = {
  hideTags: true,
  variant: "text",
};

const LABEL_LOCATION: Dialog.LocationPreference = {
  targetCorner: location.TOP_RIGHT,
  dialogCorner: location.TOP_LEFT,
};

export const MenuItem = (): ReactElement => (
  <Form.Field<label.Key[]> path="query.hasLabels" defaultValue={[]} showLabel={false}>
    {({ value, onChange }) => (
      <SelectMultiple
        value={value}
        onChange={onChange}
        location={LABEL_LOCATION}
        triggerProps={TRIGGER_PROPS}
      />
    )}
  </Form.Field>
);
