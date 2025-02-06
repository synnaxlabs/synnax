// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Button } from "@synnaxlabs/pluto";

export interface ToolbarButtonProps
  extends Omit<Button.IconProps, "onClick" | "children"> {
  onExport: NonNullable<Button.IconProps["onClick"]>;
}

export const ToolbarButton = ({ onExport, ...rest }: ToolbarButtonProps) => (
  <Button.Icon
    tooltip="Export"
    sharp
    size="medium"
    style={{ height: "100%" }}
    onClick={onExport}
    {...rest}
  >
    <Icon.Export />
  </Button.Icon>
);
