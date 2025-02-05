// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Icon, type IconFC } from "@synnaxlabs/media";
import { Align, Button, Text } from "@synnaxlabs/pluto";
import { binary } from "@synnaxlabs/x";

import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import { Link } from "@/link";

interface CopyButtonProps {
  children: IconFC;
  disabled?: boolean;
  onClick: () => void;
  tooltip: string;
}

const CopyButton = ({ children: Icon, tooltip, ...rest }: CopyButtonProps) => (
  <Button.Icon
    tooltip={() => <Text.Text level="small">{`Copy ${tooltip}`}</Text.Text>}
    tooltipLocation="left"
    variant="text"
    {...rest}
  >
    <Icon style={{ color: "var(--pluto-gray-l7)" }} />
  </Button.Icon>
);

export interface CopyButtonsProps {
  getConfig: () => unknown;
  getName: () => string;
  taskKey: task.Key;
}

export const CopyButtons = ({ getConfig, getName, taskKey }: CopyButtonsProps) => {
  const copy = useCopyToClipboard();
  const handleCopyTypeScriptCode = () => {
    const name = getName();
    copy(
      `
      // Retrieve ${name}
      const task = client.hardware.tasks.retrieve(${taskKey})
      `,
      `TypeScript code to retrieve ${name}`,
    );
  };
  const handleCopyJSONConfig = () =>
    copy(
      binary.JSON_CODEC.encodeString(getConfig()),
      `JSON configuration for ${getName()}`,
    );
  const copyLink = Link.useCopyToClipboard();
  const handleCopyLink = () =>
    copyLink({ name: getName(), ontologyID: task.ontologyID(taskKey) });
  const hasDisabledButtons = taskKey === "";
  return (
    <Align.Space direction="x" size="small">
      <CopyButton
        disabled={hasDisabledButtons}
        onClick={handleCopyTypeScriptCode}
        tooltip="TypeScript code"
      >
        {Icon.TypeScript}
      </CopyButton>
      <CopyButton onClick={handleCopyJSONConfig} tooltip="JSON configuration">
        {Icon.JSON}
      </CopyButton>
      <CopyButton disabled={hasDisabledButtons} onClick={handleCopyLink} tooltip="link">
        {Icon.Link}
      </CopyButton>
    </Align.Space>
  );
};
