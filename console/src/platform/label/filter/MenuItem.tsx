// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { type Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";

import { SelectMultiple } from "@/platform/label/Select";
import { Search } from "@/platform/search";

const TRIGGER_PROPS: Select.MultipleTriggerProps<string> = {
  hideTags: true,
  variant: "text",
};

export const MenuItem = (): ReactElement => (
  <Form.Field<label.Key[] | undefined, label.Key[]>
    path="query.hasLabels"
    showLabel={false}
  >
    {({ value, onChange }) => (
      <SelectMultiple
        value={value ?? []}
        onChange={onChange}
        location={Search.FILTER_LOCATION}
        triggerProps={TRIGGER_PROPS}
      />
    )}
  </Form.Field>
);
