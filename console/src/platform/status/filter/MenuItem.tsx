// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import {
  Component,
  Form,
  type Input,
  type position,
  type Select,
  Status,
} from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

const TRIGGER_PROPS: Select.MultipleTriggerProps<status.Variant> = {
  hideTags: true,
  variant: "text",
};

const DIALOG_LOCATION: position.LocationPreference = {
  targetCorner: location.TOP_RIGHT,
  dialogCorner: location.TOP_LEFT,
};

export const MenuItem = (): ReactElement => (
  <Form.Field<status.Variant[] | undefined, status.Variant[]>
    path="query.variants"
    showLabel={false}
  >
    {selectVariantRenderProp}
  </Form.Field>
);

const selectVariantRenderProp = Component.renderProp(
  ({
    value,
    onChange,
  }: Input.Control<status.Variant[] | undefined, status.Variant[]>) => (
    <Status.SelectMultipleVariants
      value={value ?? []}
      onChange={onChange}
      location={DIALOG_LOCATION}
      triggerProps={TRIGGER_PROPS}
    />
  ),
);
