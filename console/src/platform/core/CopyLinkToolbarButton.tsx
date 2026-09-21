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
import { type ReactElement, useCallback } from "react";

import { useCopyLinkToClipboard } from "@/platform/core/useCopyLinkToClipboard";
import { Link } from "@/platform/link";

export interface CopyLinkToolbarButtonProps extends Omit<
  Button.ButtonProps,
  "children"
> {
  name: string;
  ontologyID: ontology.ID;
}

export const CopyLinkToolbarButton = ({
  name,
  ontologyID,
  ...rest
}: CopyLinkToolbarButtonProps): ReactElement | null => {
  const disabled = Link.useDisabled();
  const copyLink = useCopyLinkToClipboard();
  const handleClick = useCallback(
    () => copyLink({ name, ontologyID }),
    [copyLink, name, ontologyID],
  );
  if (disabled) return null;
  return (
    <Button.Button
      tooltip="Copy link"
      size="medium"
      variant="text"
      onClick={handleClick}
      {...rest}
    >
      <Icon.Link />
    </Button.Button>
  );
};
