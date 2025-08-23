// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Input, Nav, Text } from "@synnaxlabs/pluto";
import { useState } from "react";

import { cleanChannelName, isAllowedChannelNameCharacter } from "@/hardware/common/task/channelNameUtils";
import { type BaseArgs, createBase, type Prompt } from "@/modals/Base";
import { ModalContentLayout } from "@/modals/layout";
import { Triggers } from "@/triggers";

export interface PromptRenameChannelsArgs extends BaseArgs<string> {
  allowEmpty?: boolean;
  initialValue?: string;
  label?: string;
  oldNames?: string[];
  currentNames?: string[];
  canRenameCmdChannel?: boolean;
  canRenameStateChannel?: boolean;
}

export const RENAME_CHANNELS_LAYOUT_TYPE = "rename-channels";

export interface PromptRenameChannels extends Prompt<string, PromptRenameChannelsArgs> {}

export const [useRenameChannels, RenameChannels] = createBase<string, PromptRenameChannelsArgs>(
  "Rename Channels",
  RENAME_CHANNELS_LAYOUT_TYPE,
  ({
    value: { result, allowEmpty = false, label, initialValue, oldNames = [], currentNames = [], canRenameCmdChannel = false, canRenameStateChannel = false },
    onFinish,
  }) => {
    const [name, setName] = useState(result ?? initialValue ?? "");
    const [error, setError] = useState<string | undefined>(undefined);
    
    const handleNameChange = (newName: string) => {
      const filteredName = newName.split('').filter(char => isAllowedChannelNameCharacter(char)).join('');
      setName(filteredName);
      setError(undefined);
    };
    
    const cleanedName = cleanChannelName(name.trim());
    
    const previewNames = [
      ...(canRenameCmdChannel ? [`${cleanedName}_cmd`, `${cleanedName}_cmd_time`] : []),
      ...(canRenameStateChannel ? [`${cleanedName}_state`] : [])
    ];
    
    const hasConflict = oldNames.some(oldName => previewNames.includes(oldName));
    const isEmpty = cleanedName === "";
    const canSubmit = !isEmpty && !hasConflict;
    
    const handleSubmit = () => {
      if (!canSubmit) {
        if (isEmpty && !allowEmpty)
          setError("Channel name cannot be empty");
        else if (hasConflict)
          setError("Channel name conflicts with existing channels");
        return;
      }
      onFinish(cleanedName);
    };

    const footer = (
      <>
        <Triggers.SaveHelpText action="Rename" trigger={Triggers.SAVE} />
        <Nav.Bar.End>
          <Button.Button
            onClick={() => onFinish(null)}
            variant="outlined"
          >
            Cancel
          </Button.Button>
          <Button.Button
            onClick={handleSubmit}
            variant="filled"
            disabled={!canSubmit}
            trigger={Triggers.SAVE}
          >
            Rename
          </Button.Button>
        </Nav.Bar.End>
      </>
    );

    return (
      <ModalContentLayout footer={footer}>
        <Input.Item
          label={label ?? "Base Channel Name"}
          required={!allowEmpty}
          helpText={error}
          status={error != null ? "error" : "success"}
          padHelpText
        >
          <Input.Text
            autoFocus
            placeholder="Enter base name"
            level="h2"
            variant="text"
            value={name}
            onChange={handleNameChange}
          />
        </Input.Item>
        {(currentNames.length > 0 || previewNames.length > 0) && (
          <Flex.Box direction="x" gap="large">
            {currentNames.length > 0 && (
              <Input.Item label="Current Names" grow={false} style={{ flex: 1, minWidth: 0 }}>
                {currentNames.map((currentName, index) => (
                  <Text.Text key={index} level="p" style={{ whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>
                    • {currentName}
                  </Text.Text>
                ))}
              </Input.Item>
            )}
            {previewNames.length > 0 && (
              <Input.Item label="New Names" grow={false} style={{ flex: 1, minWidth: 0 }}>
                {previewNames.map((previewName, index) => (
                  <Text.Text 
                    key={index} 
                    level="p" 
                    color={hasConflict ? "var(--pluto-error-m1)" : "var(--pluto-text-m1)"}
                    style={{ whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}
                  >
                    • {previewName}
                  </Text.Text>
                ))}
              </Input.Item>
            )}
          </Flex.Box>
        )}
      </ModalContentLayout>
    );
  },
);