// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Errors } from "@/errors";
import { Form } from "@/form";
import { Custom } from "@/schematic/node/common/custom";
import { ColorField } from "@/schematic/node/common/form/Color";
import { ScaleField } from "@/schematic/node/common/form/Scale";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import { type FormProps } from "@/schematic/node/spec";

interface StyleFormProps extends FormProps {
  omit?: string[];
  hideInnerOrientation?: boolean;
  hideOuterOrientation?: boolean;
  showStateOverrides?: boolean;
}

export const StyleForm = ({
  omit,
  hideInnerOrientation,
  hideOuterOrientation,
}: StyleFormProps): ReactElement => {
  const hasStateOverrides =
    Form.useFieldValue<string>("stateOverrides", { optional: true }) != null;
  return (
    <Form.Sections x>
      <Form.Section title="Label">
        <Label.TextFields omit={omit} path="label" />
      </Form.Section>
      <Form.Section title="Label placement">
        <Label.PlacementFields omit={omit} path="label" />
      </Form.Section>
      <Form.Section title="Appearance">
        {!hasStateOverrides && <ColorField path="color" optional />}
        <Form.SwitchField
          path="normallyOpen"
          label="Normally open"
          padHelpText={false}
          hideIfNull
          optional
        />
        <ScaleField path="scale" />
      </Form.Section>
      {hasStateOverrides && (
        <Form.Section title="State overrides">
          <Errors.SuspenseBoundary>
            <Custom.StateOverrideForm />
          </Errors.SuspenseBoundary>
        </Form.Section>
      )}
      <Form.Section title="Orientation">
        <Orientation.Field
          path=""
          hideInner={hideInnerOrientation}
          hideOuter={hideOuterOrientation}
          showLabel={false}
        />
      </Form.Section>
    </Form.Sections>
  );
};
