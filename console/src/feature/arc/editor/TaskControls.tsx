// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/arc/editor/TaskControls.css";

import { type rack } from "@synnaxlabs/client";
import { Arc, Button, Rack } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { useCallback, useState } from "react";

import { Arc as PlatformArc } from "@/platform/arc";
import { CSS } from "@/platform/css";
import { Task } from "@/platform/task";

const INITIAL_RACK_QUERY: rack.RetrieveParams = { integration: "arc" };

export const TaskControls = () => {
  const key = Arc.useKey();
  const name = Arc.useSelectName();
  const { running, onStartStop, taskStatus, taskRack } = PlatformArc.useTask(key, name);
  const [pickedRack, setPickedRack] = useState<rack.Key | undefined>();
  const selectedRack =
    pickedRack ?? (primitive.isNonZero(taskRack) ? taskRack : undefined);
  const [expanded, setExpanded] = useState(false);
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
          onChange={setPickedRack}
          allowNone
          location="top"
          initialQuery={INITIAL_RACK_QUERY}
        />
        <Button.Button
          onClick={handleConfigure}
          disabled={selectedRack === undefined}
          size="medium"
          variant="outlined"
          tooltip="Deploy the arc to the selected rack"
        >
          Deploy
        </Button.Button>
        <Task.Controls.StartStopButton
          running={running}
          onClick={onStartStop}
          disabled={selectedRack === undefined}
        />
      </Task.Controls.Actions>
    </Task.Controls.Frame>
  );
};
