// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form, Input, Status } from "@synnaxlabs/pluto";

import { useIsPreview } from "@/platform/task/Form";

export interface EnabledCheckboxProps {
  /** The path of the item's `disabled` flag. */
  path: string;
}

/**
 * A list row's enabled state: a status dot that toggles on click. Renders nothing
 * when the item has no disabled flag.
 */
export const EnabledCheckbox = ({ path }: EnabledCheckboxProps) => {
  const isPreview = useIsPreview();
  const { set } = Form.useContext();
  const disabled = Form.useFieldValue<boolean>(path, { optional: true });
  if (disabled == null) return null;
  return (
    <Input.Checkbox
      value={!disabled}
      onChange={(enabled) => set(path, !enabled)}
      checkedIcon={<Status.Indicator variant="success" />}
      uncheckedIcon={<Status.Indicator variant="disabled" />}
      size="small"
      disabled={isPreview}
      aria-label={disabled ? "Disabled" : "Enabled"}
      tooltip={isPreview ? undefined : disabled ? "Enable" : "Disable"}
    />
  );
};
