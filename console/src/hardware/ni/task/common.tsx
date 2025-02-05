// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Button, Text } from "@synnaxlabs/pluto";
import { binary } from "@synnaxlabs/x";

import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import { Link } from "@/link";

export interface CopyButtonsProps {
  importClass: string;
  taskKey?: string;
  getName: () => string;
  getConfig: () => any;
}

export const CopyButtons = ({
  importClass,
  taskKey,
  getName,
  getConfig,
}: CopyButtonsProps) => {
  const copy = useCopyToClipboard();
  const handleCopyPythonCode = () => {
    const name = getName();
    copy(
      `
      from synnax.hardware.ni import ${importClass}
      # Retrieve ${name}
      task = ${importClass}(client.hardware.tasks.retrieve(key=${taskKey}))
      `,
      `Python code to retrieve ${name}`,
    );
  };
  const handleCopyJsonConfig = () => {
    const name = getName();
    copy(binary.JSON_CODEC.encodeString(getConfig()), `configuration JSON for ${name}`);
  };
  const handleCopyToClipboard = Link.useCopyToClipboard();
  return (
    <Align.Space direction="x" size="small">
      {taskKey != null && (
        <Button.Icon
          tooltip={() => <Text.Text level="small">Copy Python Code</Text.Text>}
          tooltipLocation="left"
          variant="text"
          onClick={handleCopyPythonCode}
        >
          <Icon.Python style={{ color: "var(--pluto-gray-l7)" }} />
        </Button.Icon>
      )}
      <Button.Icon
        tooltip={() => <Text.Text level="small">Copy JSON Configuration</Text.Text>}
        tooltipLocation="left"
        variant="text"
        onClick={handleCopyJsonConfig}
      >
        <Icon.JSON style={{ color: "var(--pluto-gray-l7)" }} />
      </Button.Icon>
      {taskKey != null && (
        <Button.Icon
          tooltip={() => <Text.Text level="small">Copy Link</Text.Text>}
          tooltipLocation="left"
          variant="text"
          onClick={() =>
            handleCopyToClipboard({
              name: getName(),
              ontologyID: task.ontologyID(taskKey),
            })
          }
        >
          <Icon.Link style={{ color: "var(--pluto-gray-l7)" }} />
        </Button.Icon>
      )}
    </Align.Space>
  );
};
