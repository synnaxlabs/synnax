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
import { Arc, Rack } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";

import { SelectPerformance } from "@/feature/arc/editor/SelectPerformance";
import { Arc as PlatformArc } from "@/platform/arc";
import { CSS } from "@/platform/css";
import { Framer } from "@/platform/framer";
import { Task } from "@/platform/task";

const INITIAL_RACK_QUERY: rack.RetrieveMultipleParams = {
  where: { integrations: { any: ["arc"] } },
};
const RACK_TRIGGER_PROPS = {
  className: CSS.BE("rack-select", "trigger"),
  collapsible: true,
};
const RACK_DIALOG_PROPS = { className: CSS.BE("rack-select", "dialog") };

const describeChange = ({ type, path, kind }: arc.Change): string =>
  type === "automation" ? "Automation edited" : `Module ${path} ${kind}`;

/** The redeploy tooltip naming each reason, or undefined when none is known. */
const redeployTooltip = (
  changes: arc.Change[],
  rackChanged: boolean,
): string | undefined => {
  const reasons = changes.map(describeChange);
  if (rackChanged) reasons.push("Rack changed");
  if (reasons.length === 0) return undefined;
  return `${reasons.join(". ")}. Redeploy to apply.`;
};

/** The compile status of a module, which has no task to control. */
const ModuleControls = () => {
  const key = Arc.useKey();
  const stat = Arc.useModuleStatus(key);
  return (
    <Task.Controls.Bar
      className={CSS.BE("arc-editor", "controls")}
      status={stat}
      running={false}
      hideActions
    />
  );
};

const AutomationControls = () => {
  const key = Arc.useKey();
  const name = Arc.useName();
  const captureDeploy = PlatformArc.useCaptureDeploy();
  const { running, taskRack, taskPerformance, taskStatus, onStart, onStop } =
    Arc.useTaskControls(key, name, { afterSuccess: captureDeploy });
  const drifted = Arc.useDrifted({ arcKey: key });
  const tsk = Arc.useResultTask({ arcKey: key }).data;
  const changes = Arc.useChanges({ key, configHash: tsk?.configHash ?? "" });
  const rackChanged = tsk != null && tsk.status?.details.rack !== tsk.rack;
  const canControl = Framer.useCanCommand();
  const { update: updateTask } = Arc.useUpdateTask();
  const bound = primitive.isNonZero(taskRack);
  const handleRackChange = (rackKey: rack.Key) => updateTask({ key, rack: rackKey });
  const handlePerformanceChange = (performance: arc.task.Performance) =>
    updateTask({ key, performance });

  return (
    <Task.Controls.Bar
      className={CSS.BE("arc-editor", "controls")}
      status={taskStatus}
      running={running}
      drifted={drifted}
      redeployTooltip={redeployTooltip(changes, rackChanged)}
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
            location="top"
            query={INITIAL_RACK_QUERY}
            triggerProps={RACK_TRIGGER_PROPS}
            dialogProps={RACK_DIALOG_PROPS}
          />
          <SelectPerformance
            value={taskPerformance}
            onChange={handlePerformanceChange}
            disabled={!bound}
            location="top"
          />
        </>
      }
    />
  );
};

export const TaskControls = () =>
  Arc.useType() === "module" ? <ModuleControls /> : <AutomationControls />;
