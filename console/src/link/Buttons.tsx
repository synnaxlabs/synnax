// Copyright 2024 Synnax Labs, Inc.
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

import { Link } from "@/link";

export interface ToolbarCopyButtonProps extends Omit<Button.IconProps, "children"> {
  name: string;
  ontologyID: ontology.IDPayload;
}

export const ToolbarCopyButton = ({
  name,
  ontologyID,
  ...props
}: ToolbarCopyButtonProps) => {
  const handleLink = Link.useCopyToClipboard();
  return (
    <Button.Icon
      tooltip={"Copy link"}
      sharp
      size="medium"
      style={{ height: "100%", width: "var(--pluto-height-medium)" }}
      onClick={() => handleLink({ name, ontologyID })}
      {...props}
    >
      <Icon.Link />
    </Button.Icon>
  );
};
