// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import type { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { deep, type TimeStamp } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { Framer } from "@/platform/framer";
import { Bar } from "@/platform/task/controls/Bar";
import { useDrifted } from "@/platform/task/useDrifted";
import { useIsPreview } from "@/platform/task/useIsPreview";
import { useKey } from "@/platform/task/useKey";
import { useIsRunning, useStatus } from "@/platform/task/useStatus";

const path = deep.path<{
  config: { dataSavingDisabled: boolean; autoStart: boolean };
}>();

interface SettingProps {
  label: string;
  onIcon: ReactElement;
  offIcon: ReactElement;
  value: boolean;
  onChange: (value: boolean) => void;
}

const Setting = ({ label, onIcon, offIcon, value, onChange }: SettingProps) => {
  const isPreview = useIsPreview();
  return (
    <Button.Toggle
      value={value}
      onChange={onChange}
      disabled={isPreview}
      tooltip={`${label} ${value ? "on" : "off"}`}
      aria-label={label}
    >
      {value ? onIcon : offIcon}
      {value ? "On" : "Off"}
    </Button.Toggle>
  );
};

// The task-wide settings of the form, for a config that carries them.
const Settings = () => {
  const dataSaving = Form.useField(path.config.dataSavingDisabled, {
    optional: true,
  });
  const autoStart = Form.useField(path.config.autoStart, { optional: true });
  return (
    <>
      {dataSaving != null && (
        <Setting
          label="Data saving"
          onIcon={<Icon.Database />}
          offIcon={<Icon.DatabaseOff />}
          value={!dataSaving.value}
          onChange={(v) => dataSaving.onChange(!v)}
        />
      )}
      {autoStart != null && (
        <Setting
          label="Auto start"
          onIcon={<Icon.AutoStart />}
          offIcon={<Icon.AutoStartOff />}
          value={autoStart.value}
          onChange={autoStart.onChange}
        />
      )}
    </>
  );
};

/** A deploy that failed before the task took its start command. */
export interface DeployFailure {
  status: status.Status;
  /** The time of the task status when the deploy failed. Absent for no status. */
  taskStatusTime?: TimeStamp;
}

export interface ControlsProps extends Flex.BoxProps {
  /** Whether a deploy runs and has not sent its start command yet. */
  configuring?: boolean;
  /** Shown in place of the task status until the Core sends a new task status. */
  failure?: DeployFailure;
  /** Runs the deploy pipeline: configure, save the row, issue start. */
  onDeploy: () => void;
  /** Issues a stop command to the live instance. */
  onStop: () => void;
}

/** Task controls bar wired to the surrounding task Form context. */
export const Controls = ({
  configuring = false,
  failure,
  onDeploy,
  onStop,
  ...props
}: ControlsProps) => {
  const taskStatus = useStatus();
  const running = useIsRunning();
  const key = useKey();
  const drifted = useDrifted();
  const canControl = Framer.useCanCommand();

  const handleDeploy = () => {
    if (key == null) return;
    onDeploy();
  };
  const handleStop = () => {
    if (key == null) return;
    onStop();
  };

  let shownStatus: status.Status | undefined = taskStatus;
  // The start command goes out only after the config saves, which can take seconds.
  if (configuring)
    shownStatus = status.create({
      name: taskStatus?.name ?? "",
      variant: "loading",
      message: "Configuring task",
    });
  else if (
    failure != null &&
    failure.taskStatusTime?.valueOf() === taskStatus?.time.valueOf()
  )
    shownStatus = failure.status;
  let startStopVariant = status.keepVariants(taskStatus?.variant, "loading");
  // On a running task, a loading variant disables Redeploy and keeps Stop enabled.
  if (configuring) startStopVariant = running ? "loading" : undefined;

  return (
    <Bar
      status={shownStatus}
      running={running}
      drifted={drifted}
      hideActions={!canControl}
      disabled={configuring && !running}
      startStopVariant={startStopVariant}
      settings={<Settings />}
      onDeploy={handleDeploy}
      onStop={handleStop}
      {...props}
    />
  );
};
