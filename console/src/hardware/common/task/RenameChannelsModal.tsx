// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Input, Nav, Text } from "@synnaxlabs/pluto";
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
  canRenameCmdChannel?: boolean;
  canRenameStateChannel?: boolean;
}

export const RENAME_CHANNELS_LAYOUT_TYPE = "rename-channels";

export interface PromptRenameChannels extends Prompt<string, PromptRenameChannelsArgs> {}

export const [useRenameChannels, RenameChannels] = createBase<string, PromptRenameChannelsArgs>(
  "Rename Channels",
  RENAME_CHANNELS_LAYOUT_TYPE,
  ({
    value: { result, allowEmpty = false, label, initialValue, oldNames = [], canRenameCmdChannel = false, canRenameStateChannel = false },
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
    
    const footer = (
      <>
        <Triggers.SaveHelpText action="Save" trigger={Triggers.SAVE} />
        <Nav.Bar.End x align="center">
          <Button.Button
            status="success"
            disabled={!allowEmpty && name.length === 0}
            variant="filled"
            onClick={() => {
              if (name.length === 0) {
                if (allowEmpty) return onFinish(null);
                return setError(`${label} is required`);
              }
              return onFinish(cleanedName);
            }}
            trigger={Triggers.SAVE}
          >
            Rename
          </Button.Button>
        </Nav.Bar.End>
      </>
    );

    return (
      <ModalContentLayout footer={footer}>
        <div style={{ marginTop: oldNames.length === 1 ? "-8rem" : "-3rem" }}>
          <Input.Item
            label={label}
            required={!allowEmpty}
            helpText={error}
            status={error != null ? "error" : "success"}
          >
            <Input.Text
              autoFocus
              placeholder={label}
              level="h2"
              variant="text"
              value={name}
              onChange={handleNameChange}
            />
          </Input.Item>
        </div>
        
        {oldNames.length > 0 && (
          <div style={{ display: "grid", gridTemplateColumns: "200px 200px", gap: "2rem", marginBottom: "-5rem" }}>
            <div>
              <Text.Text level="h4" weight={500}>Current {oldNames.length === 1 ? "Name" : "Names"}:</Text.Text>
              {oldNames.map((oldName, index) => (
                <Text.Text key={index} level="small" color={7}>
                  {oldName}
                </Text.Text>
              ))}
            </div>
            
            <div>
              <Text.Text level="h4" weight={500}>New {previewNames.length <= 1 ? "Name" : "Names"}:</Text.Text>
              {(() => {
                if (previewNames.length > 0 && cleanedName)
                  return previewNames.map((newName, index) => (
                    <Text.Text key={index} level="small" color={7}>
                      {newName}
                    </Text.Text>
                  ));
                
                if (!canRenameCmdChannel && !canRenameStateChannel && cleanedName)
                  return (
                    <Text.Text level="small" color={7}>
                      {cleanedName}
                    </Text.Text>
                  );
                
                return (
                  <Text.Text level="small" color={7} style={{ fontStyle: "italic" }}>
                    {!canRenameCmdChannel && !canRenameStateChannel ? "Enter name" : "Enter a base name to see preview"}
                  </Text.Text>
                );
              })()}
            </div>
          </div>
        )}
      </ModalContentLayout>
    );
  },
);