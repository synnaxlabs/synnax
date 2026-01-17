// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, type rack, task } from "@synnaxlabs/client";
import {
  Arc,
  Button,
  Flex,
  type Flux,
  Icon,
  Ontology,
  Rack,
  Status,
  Task,
} from "@synnaxlabs/pluto";
import { useCallback, useEffect, useState } from "react";

import { type State } from "@/arc/slice";
import { translateGraphToServer } from "@/arc/types/translate";
import { Layout } from "@/layout";

interface ControlsProps {
  state: State;
}

export const Controls = ({ state }: ControlsProps) => {
  const name = Layout.useSelectRequiredName(state.key);
  const children = Ontology.useRetrieveChildren({
    id: arc.ontologyID(state.key),
    types: ["task"],
  });
  const tsk = Task.useRetrieve(
    { key: children.data?.[0]?.id.key ?? 0 },
    { addStatusOnFailure: false },
  );
  const [selectedRack, setSelectedRack] = useState<rack.Key | undefined>();
  useEffect(() => {
    if (tsk.data?.key == null) return;
    setSelectedRack(task.rackKey(tsk.data.key));
  }, [tsk.data?.key]);
  const { update } = Arc.useCreate({
    afterSuccess: useCallback(
      async ({ client, data }: Flux.AfterSuccessParams<arc.Arc, false>) => {
        const { key } = data;
        if (selectedRack == null) return;
        let taskKey = tsk.data?.key;
        taskKey ??= task.newKey(selectedRack, 0);
        const newTsk = await client.tasks.create({
          key: taskKey,
          name,
          type: "arc",
          config: {
            arc_key: key,
          },
        });
        if (tsk.data?.key == null)
          await client.ontology.addChildren(
            arc.ontologyID(key),
            task.ontologyID(newTsk.key),
          );

        await client.tasks.executeCommand({ task: taskKey, type: "start" });
      },
      [name, selectedRack, tsk.data?.key],
    ),
  });
  const cmd = Task.useCommand();
  const handleStop = useCallback(() => {
    if (tsk.data?.key == null) return;
    cmd.update([{ task: tsk.data.key, type: "stop" }]);
  }, [cmd, tsk.data?.key]);

  const isRunning = tsk.data?.status?.details.running ?? false;
  const handleDeploy = useCallback(() => {
    if (isRunning) handleStop();
    else
      update({
        name,
        key: state.key,
        text: state.text,
        version: "0.0.0",
        graph: translateGraphToServer(state.graph),
        mode: state.mode,
      });
  }, [state, update, handleStop, isRunning, name]);

  return (
    <Flex.Box
      style={{
        padding: "2rem",
        position: "absolute",
        bottom: 0,
        right: 0,
        width: 500,
      }}
      justify="end"
      grow
    >
      <Flex.Box
        x
        background={1}
        style={{ padding: "2rem" }}
        bordered
        borderColor={5}
        grow
        rounded={2}
        justify="between"
        gap="medium"
      >
        <Flex.Box x gap="small" align="center" grow>
          <Rack.SelectSingle
            value={selectedRack}
            onChange={setSelectedRack}
            allowNone
            style={{ minWidth: 150 }}
          />
          <Status.Summary
            variant="disabled"
            message="Not deployed"
            status={tsk.data?.status}
          />
        </Flex.Box>
        <Button.Button
          onClick={handleDeploy}
          variant="filled"
          disabled={selectedRack === undefined}
        >
          {isRunning ? <Icon.Pause /> : <Icon.Play />}
          {isRunning ? "Stop" : "Start"}
        </Button.Button>
      </Flex.Box>
    </Flex.Box>
  );
};
