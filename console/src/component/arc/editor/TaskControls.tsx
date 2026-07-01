// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/component/arc/editor/TaskControls.css";

import { type rack, task } from "@synnaxlabs/client";
import { Arc, Rack } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { useCallback, useEffect, useState } from "react";

import { useTask } from "@/component/arc/useTask";
import { CSS } from "@/component/css";
import { Task } from "@/component/task";

const INITIAL_RACK_QUERY: rack.RetrieveArgs = { integration: "arc" };

export const TaskControls = () => {
  const key = Arc.useKey();
  const name = Arc.useSelectName();
  const { running, onStartStop, taskStatus, taskKey } = useTask(key, name);
  const taskKeyDefined = primitive.isNonZero(taskKey);
  const [selectedRack, setSelectedRack] = useState<rack.Key | undefined>();
  const [expanded, setExpanded] = useState(false);

  useEffect(() => {
    if (taskKeyDefined) setSelectedRack(task.rackKey(taskKey));
  }, [taskKey, taskKeyDefined]);
  const { update } = Arc.useCreate();
  const { data: remote } = Arc.useRetrieve({ key }, { addStatusOnFailure: false });

  const handleConfigure = useCallback(() => {
    if (remote == null) return;
    update({ ...remote, name, key, rack: selectedRack });
  }, [key, update, name, selectedRack, remote]);

  const handleToggle = useCallback(() => setExpanded((prev) => !prev), []);
  const handleContract = useCallback(() => setExpanded(false), []);

  return (
    <Task.Controls.Frame
      className={CSS.BE("arc-editor", "controls")}
      expanded={expanded}
      onContract={handleContract}
    >
      <Task.Controls.Status
        status={taskStatus}
        expanded={expanded}
        onToggle={handleToggle}
        fallbackMessage="Not deployed"
      />
      <Task.Controls.Actions>
        <Rack.SelectSingle
          className={CSS.B("rack-select")}
          value={selectedRack}
          onChange={setSelectedRack}
          allowNone
          location="top"
          initialQuery={INITIAL_RACK_QUERY}
        />
        <Task.Controls.ConfigureButton
          onClick={handleConfigure}
          disabled={selectedRack === undefined}
        />
        <Task.Controls.StartStopButton
          running={running}
          onClick={onStartStop}
          disabled={selectedRack === undefined}
        />
      </Task.Controls.Actions>
    </Task.Controls.Frame>
  );
};
