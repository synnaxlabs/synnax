// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { type Flex } from "@synnaxlabs/lyra/flex";
import { Button } from "@synnaxlabs/lyra/button";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Framer } from "@/platform/framer";
import { Bar } from "@/platform/task/controls/Bar";
import { useDrifted } from "@/platform/task/useDrifted";
import { useKey } from "@/platform/task/useKey";
import { useStatus } from "@/platform/task/useStatus";
import { useFooterTaskSettings, useTaskSettingsOption } from "@/uiOptionsBar"; // SCRATCH

interface SettingProps {
  label: string;
  onIcon: ReactElement;
  offIcon: ReactElement;
  value: boolean;
  onChange: (value: boolean) => void;
}

const Setting = ({ label, onIcon, offIcon, value, onChange }: SettingProps) => {
  const word = useTaskSettingsOption() === "word"; // SCRATCH
  return (
    <Button.Toggle
      className={CSS.BE("task-controls", "setting")}
      value={value}
      onChange={onChange}
      tooltip={`${label} ${value ? "on" : "off"}`}
      aria-label={label}
    >
      {value ? onIcon : offIcon}
      {word && (value ? "On" : "Off")}
    </Button.Toggle>
  );
};

const Settings = () => {
  const dataSaving = Form.useField<boolean>("config.dataSavingDisabled", {
    optional: true,
  });
  const autoStart = Form.useField<boolean>("config.autoStart", { optional: true });
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

export interface ControlsProps extends Flex.BoxProps {
  /** Runs the deploy pipeline: configure, save the row, issue start. */
  onDeploy: () => void;
  /** Issues a stop command to the live instance. */
  onStop: () => void;
}

/** Task controls bar wired to the surrounding task Form context. */
export const Controls = ({ onDeploy, onStop, ...props }: ControlsProps) => {
  const taskStatus = useStatus();
  const isSnapshot = Form.useFieldValue<boolean>("snapshot");
  const key = useKey();
  const drifted = useDrifted();
  const canControl = Framer.useCanCommand();
  const footer = useFooterTaskSettings(); // SCRATCH

  const handleDeploy = () => {
    if (key == null) return;
    onDeploy();
  };
  const handleStop = () => {
    if (key == null) return;
    onStop();
  };

  return (
    <Bar
      status={taskStatus}
      running={taskStatus.details.running}
      drifted={drifted}
      hideActions={isSnapshot || !canControl}
      startStopVariant={status.keepVariants(taskStatus.variant, "loading")}
      onDeploy={handleDeploy}
      onStop={handleStop}
      extraActions={footer ? <Settings /> : undefined}
      {...props}
    />
  );
};
