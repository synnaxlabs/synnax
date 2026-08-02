// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/export/ToolbarButton.css";

import { type ontology } from "@synnaxlabs/client";
import { Button, Icon } from "@synnaxlabs/pluto";

import { CSS } from "@/platform/css";
import { use } from "@/platform/export/use";

export interface ToolbarButtonProps extends Omit<
  Button.ButtonProps,
  "onClick" | "children"
> {
  /** Resolves the ontology ID to export, evaluated when the button is clicked. */
  getID: () => ontology.ID;
}

export const ToolbarButton = ({ getID, ...rest }: ToolbarButtonProps) => {
  const handleExport = use();
  return (
    <Button.Button
      tooltip="Export layout"
      sharp
      size="medium"
      variant="text"
      className={CSS.B("toolbar-button")}
      onClick={() => handleExport(getID())}
      {...rest}
    >
      <Icon.Export />
    </Button.Button>
  );
};
