// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ni } from "@synnaxlabs/client";
import { unique } from "@synnaxlabs/x";
import { type z } from "zod";

import { type Device } from "@/feature/ni/device/types";
import {
  accepts,
  ANALOG_INPUT,
  ANALOG_OUTPUT,
  type Claim,
  claimsOf,
  COUNTER_INPUT,
  digitalPorts,
  lineClaims,
  type PortKind,
} from "@/feature/ni/task/ports";
import {
  AI_CHANNEL_TYPE_NAMES,
  type AIChannel,
  AO_CHANNEL_TYPE_NAMES,
  CI_CHANNEL_TYPE_NAMES,
  type DigitalChannel,
  TERMINAL_CONFIG_NAMES,
} from "@/feature/ni/task/types";
import { Task } from "@/platform/task";

const hardwareOf = (devices: Device[], key: string): ni.Hardware | undefined =>
  devices.find((d) => d.key === key)?.properties.hardware;

const conflictMessage = (port: string, claim: Claim, enabled: Claim[]): string => {
  if (claim.negative) return `Another channel uses ${port}, its negative input`;
  if (enabled.some((c) => c.negative))
    return `${port} is the negative input of another channel`;
  return `Another channel uses ${port}`;
};

// Each device has its own ports, so only the channels on one device can conflict.
const reportConflicts = <C extends Task.Channel & { device: string }>(
  kind: PortKind<C>,
  channels: C[],
  devices: Device[],
  report: Task.ReportIssue,
): void =>
  unique.unique(channels.map(({ device }) => device)).forEach((key) =>
    Task.Port.reportConflicts(
      claimsOf(
        kind,
        channels.filter(({ device }) => device === key),
        hardwareOf(devices, key),
      ),
      channels,
      report,
      { message: conflictMessage },
    ),
  );

const lower = (name: string): string => name.charAt(0).toLowerCase() + name.slice(1);

const formatRate = (hz: number): string => `${Math.floor(hz).toLocaleString()} Hz`;

const THERMOCOUPLE: AIChannel["type"] = "ai_thermocouple";

// A device that reads thermocouples adds a CJC and an autozero channel to its scan.
const SCAN_CHANNELS_PER_THERMOCOUPLE_DEVICE = 2;

const checkAnalogRate = (
  config: ni.AnalogReadConfig,
  devices: Device[],
  report: Task.ReportIssue,
): void => {
  devices.forEach(({ key, model, properties: { hardware } }) => {
    if (hardware == null) return;
    const enabled = config.channels.filter((c) => c.device === key && !c.disabled);
    if (enabled.length === 0) return;
    const single = hardware.aiMaxSingleChannelRate;
    if (single !== 0 && config.sampleRate > single)
      return report(["sampleRate"], `${model} samples at most ${formatRate(single)}`);
    const multi = hardware.aiMaxMultiChannelRate;
    if (multi === 0) return;
    let count = enabled.length;
    if (enabled.some((c) => c.type === THERMOCOUPLE))
      count += SCAN_CHANNELS_PER_THERMOCOUPLE_DEVICE;
    if (count < 2 || config.sampleRate <= multi / count) return;
    const max = formatRate(multi / count);
    report(
      ["sampleRate"],
      `${model} may not sample ${count} channels faster than ${max}`,
      "warning",
    );
  });
};

/**
 * Checks an analog read config against the ports of its devices: the ports two
 * channels on a device share, each channel's port, type, and terminal configuration,
 * and the sample rate. Only the port checks run on a device with no port data.
 */
export const analogReadHardwareZ = (devices: Device[]): z.ZodType =>
  Task.check<ni.AnalogReadConfig>((config, report) => {
    const { channels } = config;
    reportConflicts(ANALOG_INPUT, channels, devices, report);
    channels.forEach((c, i) => {
      const hardware = hardwareOf(devices, c.device);
      if (c.disabled || hardware == null || !("port" in c)) return;
      const name = `ai${c.port}`;
      const port = hardware.analogInputs.find((p) => p.port === c.port);
      if (port == null)
        return report(["channels", i, "port"], `The device has no port ${name}`);
      if (!accepts(port.types, c.type))
        report(
          ["channels", i, "type"],
          `${name} does not take ${lower(AI_CHANNEL_TYPE_NAMES[c.type])} channels`,
        );
      if (
        "terminalConfig" in c &&
        c.terminalConfig !== "Cfg_Default" &&
        !accepts(port.terminalConfigs, c.terminalConfig)
      )
        report(
          ["channels", i, "terminalConfig"],
          `${name} does not take ${lower(TERMINAL_CONFIG_NAMES[c.terminalConfig])}`,
        );
    });
    checkAnalogRate(config, devices, report);
  });

/** Checks an analog write config against the ports of its device. */
export const analogWriteHardwareZ = (device: Device | undefined): z.ZodType =>
  Task.check<ni.AnalogWriteConfig>(({ channels }, report) => {
    const hardware = device?.properties.hardware;
    Task.Port.reportConflicts(
      claimsOf(ANALOG_OUTPUT, channels, hardware),
      channels,
      report,
    );
    if (hardware == null) return;
    channels.forEach((c, i) => {
      if (c.disabled) return;
      const name = `ao${c.port}`;
      const port = hardware.analogOutputs.find((p) => p.port === c.port);
      if (port == null)
        return report(["channels", i, "port"], `The device has no port ${name}`);
      if (!accepts(port.types, c.type))
        report(
          ["channels", i, "type"],
          `${name} does not take ${lower(AO_CHANNEL_TYPE_NAMES[c.type])} channels`,
        );
    });
  });

/** Checks a counter read config against the counters of its devices. */
export const counterReadHardwareZ = (devices: Device[]): z.ZodType =>
  Task.check<ni.CounterReadConfig>(({ channels }, report) => {
    reportConflicts(COUNTER_INPUT, channels, devices, report);
    channels.forEach((c, i) => {
      const hardware = hardwareOf(devices, c.device);
      if (c.disabled || hardware == null) return;
      const name = `ctr${c.port}`;
      const port = hardware.counterInputs.find((p) => p.port === c.port);
      if (port == null)
        return report(["channels", i, "port"], `The device has no counter ${name}`);
      if (!accepts(port.types, c.type))
        report(
          ["channels", i, "type"],
          `${name} does not take ${lower(CI_CHANNEL_TYPE_NAMES[c.type])} channels`,
        );
    });
  });

/**
 * Checks a digital config against the ports of its device: the lines two channels
 * share, each line's port, and its place in the port's width.
 */
export const digitalHardwareZ = (
  device: Device | undefined,
  direction: "input" | "output",
): z.ZodType =>
  Task.check<{ channels: DigitalChannel[] }>(({ channels }, report) => {
    Task.Port.reportConflicts(lineClaims(channels), channels, report, {
      field: "line",
    });
    const hardware = device?.properties.hardware;
    if (hardware == null) return;
    const ports = digitalPorts(hardware, direction);
    channels.forEach((c, i) => {
      if (c.disabled) return;
      const port = ports.find((p) => p.port === c.port);
      if (port == null)
        return report(
          ["channels", i, "port"],
          `The device has no digital ${direction} port${c.port}`,
        );
      if (c.line >= port.lines)
        report(
          ["channels", i, "line"],
          `port${c.port} has ${port.lines} lines, numbered 0 to ${port.lines - 1}`,
        );
    });
  });
