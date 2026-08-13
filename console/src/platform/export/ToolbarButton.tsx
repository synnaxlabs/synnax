// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Icon } from "@synnaxlabs/pluto";

import { type Params,use } from "@/platform/export/use";

export interface ToolbarButtonProps extends Omit<
  Button.ButtonProps,
  "onClick" | "children"
> {
  /** Resolves the resource to export, evaluated when the button is clicked. */
  getParams: () => Params;
}

export const ToolbarButton = ({ getParams, ...rest }: ToolbarButtonProps) => {
  const handleExport = use();
  return (
    <Button.Button
      tooltip="Export layout"
      size="medium"
      variant="text"
      onClick={() => handleExport(getParams())}
      {...rest}
    >
      <Icon.Export />
    </Button.Button>
  );
};
