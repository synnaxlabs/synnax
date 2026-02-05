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
import { Arc, Rack } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { useCallback, useEffect, useState } from "react";

import { useTask } from "@/arc/hooks";
import { type State } from "@/arc/slice";
import { translateGraphToServer } from "@/arc/types/translate";
import { CSS } from "@/css";
import { Controls as Base } from "@/hardware/common/task/controls";
import { Layout } from "@/layout";

interface ControlsProps {
  state: State;
}

export const Controls = ({ state }: ControlsProps) => {
  const name = Layout.useSelectRequiredName(state.key);
  const { running, onStartStop, taskStatus, taskKey } = useTask(state.key, name);
  const taskKeyDefined = primitive.isNonZero(taskKey);
  const [selectedRack, setSelectedRack] = useState<rack.Key | undefined>();
  const [expanded, setExpanded] = useState(false);

  useEffect(() => {
    if (taskKeyDefined) setSelectedRack(task.rackKey(taskKey));
  }, [taskKey, taskKeyDefined]);
  const { update } = Arc.useCreate();

  const handleConfigure = useCallback(() => {
    update({
      name,
      key: state.key,
      text: state.text,
      version: "0.0.0",
      graph: translateGraphToServer(state.graph),
      mode: state.mode,
      rack: selectedRack,
    });
  }, [state, update, name, selectedRack]);

  const handleToggle = useCallback(() => setExpanded((prev) => !prev), []);
  const handleContract = useCallback(() => setExpanded(false), []);

  return (
    <Base.Frame
      className={CSS.BE("arc-editor", "controls")}
      expanded={expanded}
      onContract={handleContract}
    >
      <Base.Status
        status={taskStatus}
        expanded={expanded}
        onToggle={handleToggle}
        fallbackMessage="Not deployed"
      />
      <Base.Actions>
        <Rack.SelectSingle
          className={CSS.B("rack-select")}
          value={selectedRack}
          onChange={setSelectedRack}
          allowNone
          location="top"
        />
        <Base.ConfigureButton
          onClick={handleConfigure}
          disabled={selectedRack === undefined}
        />
        <Base.StartStopButton
          running={running}
          onClick={onStartStop}
          disabled={selectedRack === undefined}
        />
      </Base.Actions>
    </Base.Frame>
  );
};
