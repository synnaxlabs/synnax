// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { type color, deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Errors } from "@/errors";
import { Custom } from "@/schematic/node/common/custom";
import { ColorField } from "@/schematic/node/common/form/Color";
import { ScaleField } from "@/schematic/node/common/form/Scale";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import { type FormProps } from "@/schematic/node/spec";

const path = deep.path<StyleFormValues>();

/** The fields of a symbol that {@link StyleForm} edits. */
export interface StyleFormValues extends schematic.LabeledConfig {
  strokeColor?: color.Color;
  scale: number;
  normallyOpen?: boolean;
  stateOverrides?: schematic.symbol.State[];
}

export interface StyleFormProps extends FormProps {
  omit?: string[];
  hideInnerOrientation?: boolean;
  hideOuterOrientation?: boolean;
  showStateOverrides?: boolean;
  /** Whether the config carries `normallyOpen`. Only the solenoid valve does. */
  showNormallyOpen?: boolean;
}

export const StyleForm = ({
  actions,
  omit,
  hideInnerOrientation,
  hideOuterOrientation,
  showNormallyOpen = false,
}: StyleFormProps): ReactElement => {
  const hasStateOverrides =
    Form.useFieldValue(path.stateOverrides, { optional: true }) != null;
  return (
    <Form.Sections x>
      <Form.Section title="Label">
        <Label.Form omit={omit} path={path.label} />
      </Form.Section>
      <Form.Section title="Appearance">
        {!hasStateOverrides && <ColorField path={path.strokeColor} label="Stroke" />}
        <Form.SwitchField
          path={path.normallyOpen}
          label="Normally open"
          padHelpText={false}
          visible={showNormallyOpen}
        />
        <ScaleField path={path.scale} />
      </Form.Section>
      {hasStateOverrides && (
        <Form.Section title="State overrides" actions={actions}>
          <Errors.SuspenseBoundary>
            <Custom.StateOverrideForm />
          </Errors.SuspenseBoundary>
        </Form.Section>
      )}
      <Orientation.Section
        path={path}
        hideInner={hideInnerOrientation}
        hideOuter={hideOuterOrientation}
      />
    </Form.Sections>
  );
};
