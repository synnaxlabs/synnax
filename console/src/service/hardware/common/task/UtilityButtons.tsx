// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Button, Divider, Flex, Form, Icon } from "@synnaxlabs/pluto";
import { binary, primitive } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Cluster } from "@/cluster";
import { useExport } from "@/hardware/common/task/export";
import { useKey } from "@/hardware/common/task/useKey";

export const UtilityButtons = () => {
  const ctx = Form.useContext();
  const taskKey = useKey();
  const getName = () => ctx.get<string>("name").value;
  const export_ = useExport();
  const handleExport = () => taskKey != null && export_(taskKey);
  const getTypeScriptCode = useCallback(
    () =>
      `
      // Retrieve ${getName()}
      const task = client.tasks.retrieve("${taskKey}")
      `,
    [ctx, taskKey],
  );
  const getPythonCode = useCallback(
    () =>
      `
      # Retrieve ${getName()}
      task = client.tasks.retrieve("${taskKey}")
      `,
    [ctx, taskKey],
  );
  const getJSONConfig = useCallback(() => {
    const config = ctx.get("config").value;
    return binary.JSON_CODEC.encodeString(config);
  }, [ctx]);
  const copyLink = Cluster.useCopyLinkToClipboard();
  const handleCopyLink = () => {
    if (taskKey == null) return;
    copyLink({ name: getName(), ontologyID: task.ontologyID(taskKey) });
  };
  const hasKey = !primitive.isZero(taskKey);
  return (
    <Flex.Box x gap="small">
      {hasKey && (
        <>
          <Button.Copy
            text={getTypeScriptCode}
            tooltip="Copy TypeScript code"
            tooltipLocation="left"
            variant="text"
            successMessage={() =>
              `Copied TypeScript code for ${getName()} to clipboard`
            }
            textColor={9}
          >
            <Icon.TypeScript />
          </Button.Copy>
          <Button.Copy
            text={getPythonCode}
            tooltip="Copy Python code"
            tooltipLocation="left"
            variant="text"
            successMessage={() => `Copied Python code for ${getName()} to clipboard`}
            textColor={9}
          >
            <Icon.Python />
          </Button.Copy>
          <Divider.Divider y />
        </>
      )}
      <Button.Copy
        text={getJSONConfig}
        tooltip="Copy JSON configuration"
        tooltipLocation="left"
        variant="text"
        successMessage={() => `Copied JSON configuration for ${getName()} to clipboard`}
        textColor={9}
      >
        <Icon.JSON />
      </Button.Copy>
      {hasKey && (
        <>
          <Divider.Divider y />
          <Button.Button
            onClick={handleCopyLink}
            tooltip="Copy link"
            tooltipLocation="left"
            variant="text"
            textColor={9}
          >
            <Icon.Link />
          </Button.Button>
          <Button.Button
            onClick={handleExport}
            tooltip="Export"
            tooltipLocation="left"
            variant="text"
            textColor={9}
          >
            <Icon.Export />
          </Button.Button>
        </>
      )}
    </Flex.Box>
  );
};
