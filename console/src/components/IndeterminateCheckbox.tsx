// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback } from "react";

export interface IndeterminateCheckboxProps {
  checked: boolean;
  indeterminate?: boolean;
  onChange: () => void;
  disabled?: boolean;
}

export const IndeterminateCheckbox = ({
  checked,
  indeterminate = false,
  onChange,
  disabled = false,
}: IndeterminateCheckboxProps): ReactElement => {
  const ref = useCallback(
    (el: HTMLInputElement | null) => {
      if (el != null) el.indeterminate = indeterminate;
    },
    [indeterminate],
  );
  return (
    <input
      type="checkbox"
      ref={ref}
      checked={checked}
      onChange={onChange}
      onClick={(e) => e.stopPropagation()}
      disabled={disabled}
      style={{ marginRight: "0.25rem" }}
    />
  );
};
