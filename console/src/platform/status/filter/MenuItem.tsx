// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Form } from "@synnaxlabs/lyra/form";
import { type Input } from "@synnaxlabs/lyra/input";
import { type position } from "@synnaxlabs/lyra/position";
import { type Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { getSelected } from "@/platform/status/filter/selected";

const TRIGGER_PROPS: Select.MultipleTriggerProps<status.Variant> = {
  hideTags: true,
  variant: "text",
};

const DIALOG_LOCATION: position.LocationPreference = {
  targetCorner: location.TOP_RIGHT,
  dialogCorner: location.TOP_LEFT,
};

type VariantSet = status.Where["variant"];

export const MenuItem = (): ReactElement => (
  <Form.Field<VariantSet> path="where.variant" showLabel={false} showHelpText={false}>
    {selectVariantRenderProp}
  </Form.Field>
);

const selectVariantRenderProp = Component.renderProp(
  ({ value, onChange }: Input.Control<VariantSet>) => (
    <Status.SelectMultipleVariants
      value={getSelected(value)}
      // An empty in matches nothing, so no selection drops the line.
      onChange={(variants) =>
        onChange(variants.length === 0 ? undefined : { in: variants })
      }
      location={DIALOG_LOCATION}
      triggerProps={TRIGGER_PROPS}
    />
  ),
);
