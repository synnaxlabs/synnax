// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Nav, Text } from "@synnaxlabs/pluto";

import { type BaseArgs, createBase, type Prompt } from "@/modals/Base";
import { ModalContentLayout } from "@/modals/layout";
import { Triggers } from "@/triggers";

export interface PromptDeleteChannelsArgs extends BaseArgs<boolean> {
  message: string;
  channelNames: string[];
}

export const DELETE_CHANNELS_LAYOUT_TYPE = "delete-channels";

export interface PromptDeleteChannels extends Prompt<boolean, PromptDeleteChannelsArgs> {}

export const [useDeleteChannels, DeleteChannels] = createBase<boolean, PromptDeleteChannelsArgs>(
  "Delete Channels",
  DELETE_CHANNELS_LAYOUT_TYPE,
  ({
    value: { message, channelNames },
    onFinish,
  }) => {
    const ITEMS_PER_COLUMN = 3;
    const footer = (
      <>
        <Triggers.SaveHelpText action="Delete" trigger={Triggers.SAVE} />
        <Nav.Bar.End x align="center">
          <Button.Button
            onClick={() => onFinish(false)}
          >
            Cancel
          </Button.Button>
          <Button.Button
            variant="filled"
            status="error"
            onClick={() => onFinish(true)}
            trigger={Triggers.SAVE}
          >
            Delete
          </Button.Button>
        </Nav.Bar.End>
      </>
    );

    return (
      <ModalContentLayout footer={footer}>
        <Text.Text level="h3" weight={450}>
          {message}
        </Text.Text>
        <Text.Text weight={450}>This action cannot be undone.</Text.Text>
        
        {channelNames.length > 0 && (
          <div style={{ marginBottom: "-5rem"}}>
            {channelNames.length <= ITEMS_PER_COLUMN ? (
              // Single column for few items
              channelNames.map((channelName, index) => (
                <Text.Text 
                  key={index} 
                  level="small" 
                  color={7}
                  style={{ fontStyle: "italic", display: "block", marginBottom: "0rem" }}
                >
                  {channelName}
                </Text.Text>
              ))
            ) : (
              // Multiple columns for many items
              <Flex.Box x gap="large">
                {Array.from({ length: Math.ceil(channelNames.length / ITEMS_PER_COLUMN) }).map((_, colIndex) => (
                  <Flex.Box key={colIndex} y gap="tiny" style={{ flex: "0 0 auto" }}>
                    {channelNames
                      .slice(colIndex * ITEMS_PER_COLUMN, (colIndex + 1) * ITEMS_PER_COLUMN)
                      .map((channelName, index) => (
                        <Text.Text 
                          key={colIndex * ITEMS_PER_COLUMN + index} 
                          level="small" 
                          color={7}
                          style={{ fontStyle: "italic" }}
                        >
                          {channelName}
                        </Text.Text>
                      ))
                    }
                  </Flex.Box>
                ))}
              </Flex.Box>
            )}
          </div>
        )}
      </ModalContentLayout>
    );
  },
);