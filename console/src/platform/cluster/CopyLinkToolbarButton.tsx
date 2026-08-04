// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { useCopyLinkToClipboard } from "@/platform/cluster/useCopyLinkToClipboard";
import { Toolbar } from "@/platform/toolbar";

export interface CopyLinkToolbarButtonProps extends Omit<
  Toolbar.ButtonProps,
  "children"
> {
  name: string;
  ontologyID: ontology.ID;
}

export const CopyLinkToolbarButton = ({
  name,
  ontologyID,
  ...rest
}: CopyLinkToolbarButtonProps): ReactElement => {
  const copyLink = useCopyLinkToClipboard();
  const handleClick = useCallback(
    () => copyLink({ name, ontologyID }),
    [copyLink, name, ontologyID],
  );
  return (
    <Toolbar.Button tooltip="Copy link" onClick={handleClick} {...rest}>
      <Icon.Link />
    </Toolbar.Button>
  );
};
