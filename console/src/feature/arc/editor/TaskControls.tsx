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
import { Arc, Rack } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Arc as PlatformArc } from "@/platform/arc";
import { CSS } from "@/platform/css";
import { Task } from "@/platform/task";

const INITIAL_RACK_QUERY: rack.RetrieveParams = { integration: "arc" };

export const TaskControls = () => {
  const key = Arc.useKey();
  const name = Arc.useSelectName();
  const { running, taskRack, taskStatus, onDeploy, onStop } = PlatformArc.useTask(
    key,
    name,
  );
  const drifted = PlatformArc.useDrifted(key);
  const { update: deploy } = Arc.useDeploy();

  const handleRackChange = useCallback(
    (rackKey: rack.Key | undefined) => deploy({ key, rack: rackKey ?? 0 }),
    [deploy, key],
  );

  return (
    <Task.Controls.Bar
      className={CSS.BE("arc-editor", "controls")}
      status={taskStatus}
      running={running}
      drifted={drifted}
      disabled={!primitive.isNonZero(taskRack)}
      onDeploy={onDeploy}
      onStop={onStop}
      extraActions={
        <Rack.SelectSingle
          className={CSS.B("rack-select")}
          value={primitive.isNonZero(taskRack) ? taskRack : undefined}
          onChange={handleRackChange}
          allowNone={!running}
          location="top"
          initialQuery={INITIAL_RACK_QUERY}
        />
      }
    />
  );
};
