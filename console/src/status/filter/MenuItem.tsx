// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component, type Dialog, Form, type Select, Status } from "@synnaxlabs/pluto";
import { location, type status } from "@synnaxlabs/x";
import { type ReactElement } from "react";

const TRIGGER_PROPS: Select.MultipleTriggerProps<status.Variant> = {
  hideTags: true,
  variant: "text",
};

const DIALOG_LOCATION: Dialog.LocationPreference = {
  targetCorner: location.TOP_RIGHT,
  dialogCorner: location.TOP_LEFT,
};

export const MenuItem = (): ReactElement => (
  <Form.Field<status.Variant[]>
    path="query.variants"
    defaultValue={DEFAULT_VALUE}
    showLabel={false}
  >
    {selectVariantRenderProp}
  </Form.Field>
);

const DEFAULT_VALUE: status.Variant[] = [];

const selectVariantRenderProp = Component.renderProp(
  (
    props: Pick<
      Select.MultipleProps<status.Variant, Select.StaticEntry<status.Variant>>,
      "value" | "onChange"
    >,
  ) => (
    <Status.SelectVariantMultiple
      {...props}
      location={DIALOG_LOCATION}
      triggerProps={TRIGGER_PROPS}
    />
  ),
);
