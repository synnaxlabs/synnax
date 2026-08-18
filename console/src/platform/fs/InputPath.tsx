// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Input, Status } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { PickerRow, type PickerRowProps } from "@/platform/fs/PickerRow";
import { Runtime } from "@/platform/runtime";

export interface InputPathProps
  extends Input.Control<string>, Omit<PickerRowProps, "value" | "onClick"> {
  filters?: Runtime.FileFilter[];
}

/**
 * An input holding an absolute file path chosen through the native picker. Only the
 * desktop app can produce a path, so the picker rejects in the browser.
 */
export const InputPath = ({
  value,
  onChange,
  filters,
  ...rest
}: InputPathProps): ReactElement => {
  const handleError = Status.useErrorHandler();
  const handleClick = () =>
    handleError(async () => {
      const path = await Runtime.pickPath({ filters });
      if (path == null) return;
      onChange(path);
    }, "Failed to open file");
  return <PickerRow value={value} onClick={handleClick} {...rest} />;
};
