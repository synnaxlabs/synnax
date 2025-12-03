// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Button, Flex, Form, Icon, Text } from "@synnaxlabs/pluto";
import { binary } from "@synnaxlabs/x";

import { Cluster } from "@/cluster";
import { useExport } from "@/hardware/common/task/export";
import { useKey } from "@/hardware/common/task/useKey";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";

interface UtilityButtonProps {
  children: Icon.FC;
  disabled?: boolean;
  onClick: () => void;
  tooltip: string;
}

const UtilityButton = ({ children: Icon, tooltip, ...rest }: UtilityButtonProps) => (
  <Button.Button
    tooltip={<Text.Text level="small">{tooltip}</Text.Text>}
    tooltipLocation="left"
    variant="text"
    {...rest}
  >
    <Icon style={{ color: "var(--pluto-gray-l9)" }} />
  </Button.Button>
);

export const UtilityButtons = () => {
  const ctx = Form.useContext();
  const taskKey = useKey();
  const getName = () => ctx.get<string>("name").value;
  const copy = useCopyToClipboard();
  const export_ = useExport();
  const handleExport = () => taskKey != null && export_(taskKey);
  const handleCopyTypeScriptCode = () => {
    const name = getName();
    copy(
      `
      // Retrieve ${name}
      const task = client.hardware.tasks.retrieve("${taskKey}")
      `,
      `TypeScript code for retrieving ${name}`,
    );
  };
  const handleCopyPythonCode = () => {
    const name = getName();
    copy(
      `
      # Retrieve ${name}
      task = client.hardware.tasks.retrieve("${taskKey}")
      `,
      `Python code for retrieving ${name}`,
    );
  };
  const handleCopyJSONConfig = () => {
    const name = getName();
    const config = ctx.get("config").value;
    copy(binary.JSON_CODEC.encodeString(config), `JSON configuration for ${name}`);
  };
  const copyLink = Cluster.useCopyLinkToClipboard();
  const handleCopyLink = () => {
    const name = getName();
    if (taskKey == null) return;
    copyLink({ name, ontologyID: task.ontologyID(taskKey) });
  };
  const hasDisabledButtons = taskKey === "";
  return (
    <Flex.Box x empty>
      <UtilityButton
        disabled={hasDisabledButtons}
        onClick={handleCopyTypeScriptCode}
        tooltip="Copy TypeScript code"
      >
        {Icon.TypeScript}
      </UtilityButton>
      <UtilityButton onClick={handleCopyPythonCode} tooltip="Copy Python code">
        {Icon.Python}
      </UtilityButton>
      <UtilityButton onClick={handleCopyJSONConfig} tooltip="Copy JSON configuration">
        {Icon.JSON}
      </UtilityButton>
      <UtilityButton
        onClick={handleExport}
        disabled={hasDisabledButtons}
        tooltip="Export"
      >
        {Icon.Export}
      </UtilityButton>
      <UtilityButton
        disabled={hasDisabledButtons}
        onClick={handleCopyLink}
        tooltip="Copy link"
      >
        {Icon.Link}
      </UtilityButton>
    </Flex.Box>
  );
};
