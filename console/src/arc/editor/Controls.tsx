// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/arc/editor/Controls.css";

import { type rack, task } from "@synnaxlabs/client";
import { Arc, Button, Flex, Icon, Rack, Status } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { useCallback, useEffect, useState } from "react";

import { useTask } from "@/arc/hooks";
import { type State } from "@/arc/slice";
import { translateGraphToServer } from "@/arc/types/translate";
import { CSS } from "@/css";
import { Layout } from "@/layout";

interface ControlsProps {
  state: State;
}

export const Controls = ({ state }: ControlsProps) => {
  const name = Layout.useSelectRequiredName(state.key);
  const { running, onStartStop, taskStatus, taskKey } = useTask(state.key, name);
  const taskKeyDefined = primitive.isNonZero(taskKey);
  const [selectedRack, setSelectedRack] = useState<rack.Key | undefined>();
  useEffect(() => {
    if (taskKeyDefined) setSelectedRack(task.rackKey(taskKey));
  }, [taskKey]);
  const { update } = Arc.useCreate();

  const handleConfigure = useCallback(() => {
    update({
      name,
      key: state.key,
      text: state.text,
      graph: translateGraphToServer(state.graph),
      mode: state.mode,
      rack: selectedRack,
    });
  }, [state, update, name, selectedRack]);
  return (
    <Flex.Box
      className={CSS.BE("arc-editor", "controls")}
      justify="between"
      grow
      x
      background={0}
      borderColor={5}
      bordered
      rounded={1}
    >
      <Status.Summary variant="disabled" message="Not deployed" status={taskStatus} />
      <Flex.Box x gap="small" align="center">
        <Rack.SelectSingle
          className={CSS.B("rack-select")}
          value={selectedRack}
          onChange={setSelectedRack}
          allowNone
          location="top"
        />
        <Button.Button
          onClick={handleConfigure}
          variant="outlined"
          disabled={selectedRack === undefined}
        >
          Configure
        </Button.Button>
        <Button.Button
          onClick={onStartStop}
          variant="filled"
          disabled={selectedRack === undefined}
        >
          {running ? <Icon.Pause /> : <Icon.Play />}
        </Button.Button>
      </Flex.Box>
    </Flex.Box>
  );
};
