// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type resource } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement } from "react";

import { useCopyLinkToClipboard } from "@/platform/core/useCopyLinkToClipboard";
import { Session } from "@/session";

export interface CopyLinkToolbarButtonProps extends Omit<
  Button.ButtonProps,
  "children"
> {
  name: string;
  resourceID: resource.ID;
}

export const CopyLinkToolbarButton = ({
  name,
  resourceID,
  ...rest
}: CopyLinkToolbarButtonProps): ReactElement | null => {
  const copyLink = useCopyLinkToClipboard();
  const handleClick = () => copyLink({ name, resourceID });
  if (Session.Runtime.LINKS_DISABLED) return null;
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
