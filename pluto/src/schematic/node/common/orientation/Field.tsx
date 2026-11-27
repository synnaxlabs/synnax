// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select as Base } from "@synnaxlabs/lyra/select";
import type { location } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import type { Label } from "@/schematic/node/common/label";
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
    <Form.Field label="Layout" padHelpText={false} {...rest}>
      {({ value, onChange }) =>
        hideOuter ? (
          <Base.Buttons
            value={value.orientation ?? "top"}
            onChange={(orientation: location.Outer) =>
              onChange({ ...value, orientation })
            }
          >
            <Base.Item itemKey="left" tooltip="Left" aria-label="Left">
              <Icon.Arrow.Left />
            </Base.Item>
            <Base.Item itemKey="top" tooltip="Top" aria-label="Top">
              <Icon.Arrow.Up />
            </Base.Item>
            <Base.Item itemKey="right" tooltip="Right" aria-label="Right">
              <Icon.Arrow.Right />
            </Base.Item>
            <Base.Item itemKey="bottom" tooltip="Bottom" aria-label="Bottom">
              <Icon.Arrow.Down />
            </Base.Item>
          </Base.Buttons>
        ) : (
          <Select
            value={{
              inner: value.orientation ?? "top",
              outer: value.label.orientation,
            }}
            hideInner={hideInner}
            showOuterCenter={showOuterCenter}
            onChange={(v) =>
              onChange({
                ...value,
                orientation: v.inner,
                label: { ...value.label, orientation: v.outer },
              })
            }
          />
        )
      }
    </Form.Field>
  );
};
