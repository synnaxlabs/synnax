// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { location } from "@synnaxlabs/x/location";
import { type ReactElement } from "react";

import { Form } from "@synnaxlabs/lyra/form";
import { type Label } from "@/schematic/node/common/label";
import { Select } from "@/schematic/node/common/orientation/select";

interface SymbolOrientation {
  label?: Label.Config;
  orientation?: location.Outer;
}

interface FieldExtraProps {
  hideOuter?: boolean;
  hideInner?: boolean;
  showOuterCenter?: boolean;
}

export const Field = ({
  hideOuter,
  hideInner,
  showOuterCenter,
  ...rest
}: Form.FieldProps<SymbolOrientation> & FieldExtraProps): ReactElement | null => {
  if (hideInner && hideOuter) return null;
  return (
    <Form.Field<SymbolOrientation>
      label="Orientation"
      padHelpText={false}
      required={false}
      {...rest}
    >
      {({ value, onChange }) => (
        <Select
          value={{
            inner: value.orientation ?? "top",
            outer: value.label?.orientation ?? "top",
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
