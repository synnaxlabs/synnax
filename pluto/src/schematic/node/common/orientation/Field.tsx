// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Form } from "@/form";
import { type Label } from "@/schematic/node/common/label";
import { Select } from "@/schematic/node/common/orientation/select";

interface SymbolOrientation {
  label: Label.Config;
  orientation?: location.Outer;
}

interface FieldExtraProps {
  hideOuter?: boolean;
  hideInner?: boolean;
  showOuterCenter?: boolean;
}

export type FieldProps = Form.FieldProps<SymbolOrientation> & FieldExtraProps;

export const Field = ({
  hideOuter,
  hideInner,
  showOuterCenter,
  ...rest
}: FieldProps): ReactElement | null => {
  if (hideInner && hideOuter) return null;
  return (
    <Form.Field<SymbolOrientation>
      label="Layout"
      padHelpText={false}
      required={false}
      {...rest}
    >
      {({ value, onChange }) => (
        <Select
          value={{
            inner: value.orientation ?? "top",
            outer: value.label.orientation,
          }}
          hideInner={hideInner}
          hideOuter={hideOuter}
          showOuterCenter={showOuterCenter}
          onChange={(v) =>
            onChange({
              ...value,
              orientation: v.inner,
              label: { ...value.label, orientation: v.outer },
            })
          }
        />
      )}
    </Form.Field>
  );
};

/** Section is the field in a titled form section, or nothing when both parts hide. */
export const Section = (props: FieldProps): ReactElement | null => {
  if (props.hideInner && props.hideOuter) return null;
  return (
    <Form.Section title="Layout">
      <Field showLabel={false} {...props} />
    </Form.Section>
  );
};
