// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/arc/editor/TaskControls.css";

import { type arc, type rack } from "@synnaxlabs/client";
import { Select } from "@synnaxlabs/lyra/select";
import { Arc, Rack } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Arc as PlatformArc } from "@/platform/arc";
import { CSS } from "@/platform/css";
import { Framer } from "@/platform/framer";
import { Task } from "@/platform/task";

const INITIAL_RACK_QUERY: rack.RetrieveParams = { integration: "arc" };

const PERFORMANCE_ITEMS = (
  <>
    <Select.Item<arc.task.Performance> itemKey="auto">Auto</Select.Item>
    <Select.Item<arc.task.Performance> itemKey="low">Low: least CPU</Select.Item>
    <Select.Item<arc.task.Performance> itemKey="medium">Medium: some CPU</Select.Item>
    <Select.Item<arc.task.Performance> itemKey="high">
      High: one full CPU core
    </Select.Item>
  </>
);

export const TaskControls = () => {
  const key = Arc.useKey();
  const name = Arc.useName();
  const captureDeploy = PlatformArc.useCaptureDeploy();
  const { running, taskRack, taskPerformance, taskStatus, onStart, onStop } =
    Arc.useTaskControls(key, name, { afterSuccess: captureDeploy });
  const drifted = Arc.useDrifted({ arcKey: key });
  const canControl = Framer.useCanCommand();
  const { update: updateTask } = Arc.useUpdateTask();
  const bound = primitive.isNonZero(taskRack);

  const handleRackChange = useCallback(
    (rackKey: rack.Key | undefined) => updateTask({ key, rack: rackKey ?? 0 }),
    [updateTask, key],
  );
  const handlePerformanceChange = useCallback(
    (performance: arc.task.Performance) => updateTask({ key, performance }),
    [updateTask, key],
  );

  return (
    <Task.Controls.Bar
      className={CSS.BE("arc-editor", "controls")}
      status={taskStatus}
      running={running}
      drifted={drifted}
      hideActions={!canControl}
      disabled={!bound}
      onDeploy={onStart}
      onStop={onStop}
      extraActions={
        <>
          <Rack.SelectSingle
            className={CSS.B("rack-select")}
            value={bound ? taskRack : undefined}
            onChange={handleRackChange}
            allowNone={!running}
            location="top"
            initialQuery={INITIAL_RACK_QUERY}
          />
          <Select.Simple<arc.task.Performance>
            className={CSS.B("performance-select")}
            value={taskPerformance}
            onChange={handlePerformanceChange}
            allowNone={false}
            disabled={!bound}
            location="top"
            resourceName="performance"
          >
            {PERFORMANCE_ITEMS}
          </Select.Simple>
        </>
      }
    />
  );
};
