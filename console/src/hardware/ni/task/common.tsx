// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Button, Device, Form, Synnax, Text } from "@synnaxlabs/pluto";
import { binary } from "@synnaxlabs/x";

import { Device as NIDevice } from "@/hardware/ni/device";
import { type Properties } from "@/hardware/ni/device/types";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import { Layout } from "@/layout";
import { Link } from "@/link";

export const SelectDevice = () => {
  const client = Synnax.use();
  const place = Layout.usePlacer();
  const handleDeviceChange = (v: string) => {
    if (client == null) return;
    void client.hardware.devices
      .retrieve<Properties>(v)
      .then(({ configured }) => {
        if (configured) return;
        place(NIDevice.createConfigureLayout(v, {}));
      })
      .catch(console.error);
  };
  return (
    <Form.Field<string>
      path="config.device"
      label="Device"
      grow
      onChange={handleDeviceChange}
      style={{ width: "100%" }}
    >
      {(p) => (
        <Device.SelectSingle
          allowNone={false}
          grow
          {...p}
          autoSelectOnNone={false}
          searchOptions={{ makes: ["NI"] }}
        />
      )}
    </Form.Field>
  );
};

export interface UseCopyRetrievalCodeProps {
  importClass: string;
  taskKey?: string;
  getName: () => string;
  getConfig: () => any;
}

export interface UseCopyRetrievalCodeReturn {
  copyPython: () => void;
  copyJSON: () => void;
}

export const useCopyUtils = ({
  importClass,
  taskKey,
  getName,
  getConfig,
}: UseCopyRetrievalCodeProps): UseCopyRetrievalCodeReturn => {
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

  return { copyPython: handleCopyPythonCode, copyJSON: handleCopyJsonConfig };
};

export const CopyButtons = (props: UseCopyRetrievalCodeProps) => {
  const { getName, taskKey } = props;
  const { copyPython, copyJSON } = useCopyUtils(props);
  const handleCopyToClipBoard = Link.useCopyToClipboard();
  return (
    <Align.Space direction="x" size="small">
      {taskKey != null && (
        <Button.Icon
          tooltip={() => <Text.Text level="small">Copy Python Code</Text.Text>}
          tooltipLocation="left"
          variant="text"
          onClick={copyPython}
        >
          <Icon.Python style={{ color: "var(--pluto-gray-l7)" }} />
        </Button.Icon>
      )}
      <Button.Icon
        tooltip={() => <Text.Text level="small">Copy JSON Configuration</Text.Text>}
        tooltipLocation="left"
        variant="text"
        onClick={copyJSON}
      >
        <Icon.JSON style={{ color: "var(--pluto-gray-l7)" }} />
      </Button.Icon>
      {taskKey != null && (
        <Button.Icon
          tooltip={() => <Text.Text level="small">Copy Link</Text.Text>}
          tooltipLocation="left"
          variant="text"
          onClick={() =>
            handleCopyToClipBoard({
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
