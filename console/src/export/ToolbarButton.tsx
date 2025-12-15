// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Icon } from "@synnaxlabs/pluto";

export interface ToolbarButtonProps
  extends Omit<Button.ButtonProps, "onClick" | "children"> {
  onExport: NonNullable<Button.ButtonProps["onClick"]>;
}

export const ToolbarButton = ({ onExport, ...rest }: ToolbarButtonProps) => (
  <Button.Button
    tooltip="Export layout"
    sharp
    size="medium"
    variant="text"
    style={{ height: "100%" }}
    onClick={onExport}
    {...rest}
  >
    <Icon.Export />
  </Button.Button>
);
