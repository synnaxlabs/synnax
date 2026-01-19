// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/arc/editor/Controls.css";

import { arc, type rack, task } from "@synnaxlabs/client";
import { Arc, Button, Flex, type Flux, Icon, Rack, Status } from "@synnaxlabs/pluto";
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
  const { running, onStartStop, taskStatus: status, taskKey } = useTask(state.key);
  const taskKeyDefined = primitive.isNonZero(taskKey);
  const [selectedRack, setSelectedRack] = useState<rack.Key | undefined>();
  useEffect(() => {
    if (taskKeyDefined) setSelectedRack(task.rackKey(taskKey));
  }, [taskKey]);
  const { update } = Arc.useCreate({
    afterSuccess: useCallback(
      async ({ client, data }: Flux.AfterSuccessParams<arc.Arc, false>) => {
        const { key } = data;
        if (selectedRack == null) return;
        let taskKeyToUse = taskKey;
        if (!taskKeyDefined) taskKeyToUse = task.newKey(selectedRack, 0);
        const newTsk = await client.tasks.create({
          key: taskKeyToUse,
          name,
          type: "arc",
          config: { arcKey: key },
        });
        if (taskKeyDefined)
          await client.ontology.addChildren(
            arc.ontologyID(key),
            task.ontologyID(newTsk.key),
          );
      },
      [name, selectedRack, taskKey],
    ),
  });

  const handleConfigure = useCallback(() => {
    update({
      name,
      key: state.key,
      text: state.text,
      version: "0.0.0",
      graph: translateGraphToServer(state.graph),
      mode: state.mode,
    });
  }, [state, update, name]);
  return (
    <Flex.Box
      className={CSS.BE("arc-editor", "controls")}
      justify="end"
      grow
      x
      background={0}
      borderColor={5}
      bordered
      rounded={1}
    >
      <Status.Summary variant="disabled" message="Not deployed" status={status} grow />
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
          variant="filled"
          disabled={selectedRack === undefined}
        >
          Configure
        </Button.Button>
        <Button.Button
          onClick={onStartStop}
          variant="outlined"
          disabled={selectedRack === undefined}
        >
          {running ? <Icon.Pause /> : <Icon.Play />}
        </Button.Button>
      </Flex.Box>
    </Flex.Box>
  );
};
