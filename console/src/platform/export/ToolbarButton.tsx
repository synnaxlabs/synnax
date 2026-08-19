// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { Button, Icon } from "@synnaxlabs/pluto";

import { useResource } from "@/platform/export/use";

export interface ToolbarButtonProps extends Omit<
  Button.ButtonProps,
  "onClick" | "children" | "id"
> {
  /** The ontology ID to export. */
  id: ontology.ID;
}

export const ToolbarButton = ({ id, ...rest }: ToolbarButtonProps) => {
  const handleExport = useResource();
  return (
    <Button.Button
      tooltip="Export layout"
      size="medium"
      variant="text"
      onClick={() => handleExport(id)}
      {...rest}
    >
      <Icon.Export />
    </Button.Button>
  );
};
