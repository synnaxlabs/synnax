// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Button } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { useCopyLinkToClipboard } from "@/cluster/useCopyLinkToClipboard";

export interface CopyLinkToolbarButtonProps extends Omit<Button.IconProps, "children"> {
  name: string;
  ontologyID: ontology.IDPayload;
}

export const CopyLinkToolbarButton = ({
  name,
  ontologyID,
  ...rest
}: CopyLinkToolbarButtonProps) => {
  const copyLink = useCopyLinkToClipboard();
  const handleClick = useCallback(
    () => copyLink({ name, ontologyID }),
    [copyLink, name, ontologyID],
  );
  return (
    <Button.Icon
      tooltip="Copy link"
      sharp
      size="medium"
      style={{ height: "100%", width: "var(--pluto-height-medium)" }}
      onClick={handleClick}
      {...rest}
    >
      <Icon.Link />
    </Button.Icon>
  );
};
