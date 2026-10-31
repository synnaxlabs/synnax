// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ni } from "@synnaxlabs/client";
import { z } from "zod";

import { type Device } from "@/feature/ni/device/types";
import {
  accepts,
  analogInputClaims,
  type Claims,
  conflicts,
  digitalPorts,
  lineClaims,
  portClaims,
} from "@/feature/ni/task/ports";
import {
  AI_CHANNEL_TYPE_NAMES,
  type AIChannel,
  AO_CHANNEL_TYPE_NAMES,
  CI_CHANNEL_TYPE_NAMES,
  type DigitalChannel,
  TERMINAL_CONFIG_NAMES,
} from "@/feature/ni/task/types";

type Path = PropertyKey[];

interface Report {
  (path: Path, message: string, variant?: "warning"): void;
}

const check = <T>(fn: (value: T, report: Report) => void): z.ZodType =>
  z.custom<T>().check(({ value, issues }) =>
    fn(value, (path, message, variant) =>
      issues.push({
        code: "custom",
        path,
        message,
        input: value,
        params: variant == null ? undefined : { variant },
      }),
    ),
  );

const hardwareOf = (devices: Device[], key: string): ni.Hardware | undefined =>
  devices.find((d) => d.key === key)?.properties.hardware;

const reportConflicts = (
  claims: Claims,
  prefix: string,
  indexOf: Map<string, number>,
  report: Report,
): void =>
  claims.forEach((list, port) => {
    if (!conflicts(list)) return;
    const name = `${prefix}${port}`;
    const active = list.filter((c) => !c.disabled);
    active.forEach(({ channel, negative }) => {
      const i = indexOf.get(channel);
      if (i == null) return;
      let message = `Another channel uses ${name}`;
      if (negative) message = `Another channel uses ${name}, its negative input`;
      else if (active.some((c) => c.negative))
        message = `${name} is the negative input of another channel`;
      report(["channels", i, "port"], message);
    });
  });

const indexes = (channels: { key: string }[]): Map<string, number> =>
  new Map(channels.map(({ key }, i) => [key, i]));

const lower = (name: string): string => name.charAt(0).toLowerCase() + name.slice(1);

const formatRate = (hz: number): string => `${Math.floor(hz).toLocaleString()} Hz`;

const THERMOCOUPLE: AIChannel["type"] = "ai_thermocouple";

// A device that reads thermocouples adds a CJC and an autozero channel to its scan.
const SCAN_CHANNELS_PER_THERMOCOUPLE_DEVICE = 2;

const checkAnalogRate = (
  config: ni.AnalogReadConfig,
  devices: Device[],
  report: Report,
): void => {
  devices.forEach(({ key, model, properties: { hardware } }) => {
    if (hardware == null) return;
    const enabled = config.channels.filter((c) => c.device === key && !c.disabled);
    if (enabled.length === 0) return;
    const single = hardware.aiMaxSingleChannelRate;
    if (single != null && config.sampleRate > single)
      return report(["sampleRate"], `${model} samples at most ${formatRate(single)}`);
    const multi = hardware.aiMaxMultiChannelRate;
    if (multi == null) return;
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
 * Checks an analog read config against the ports of its devices: each channel's port,
 * type, and terminal configuration, the ports two channels share, and the sample rate.
 * A device with no port data passes.
 */
export const analogReadHardwareZ = (devices: Device[]): z.ZodType =>
  check<ni.AnalogReadConfig>((config, report) => {
    const { channels } = config;
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
    const indexOf = indexes(channels);
    devices.forEach(({ key, properties: { hardware } }) => {
      if (hardware == null) return;
      reportConflicts(
        analogInputClaims(channels, key, hardware),
        "ai",
        indexOf,
        report,
      );
    });
    checkAnalogRate(config, devices, report);
  });

/** Checks an analog write config against the ports of its device. */
export const analogWriteHardwareZ = (device: Device | undefined): z.ZodType =>
  check<ni.AnalogWriteConfig>(({ channels }, report) => {
    const hardware = device?.properties.hardware;
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
    reportConflicts(portClaims(channels), "ao", indexes(channels), report);
  });

/** Checks a counter read config against the counters of its devices. */
export const counterReadHardwareZ = (devices: Device[]): z.ZodType =>
  check<ni.CounterReadConfig>(({ channels }, report) => {
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
    const indexOf = indexes(channels);
    devices.forEach(({ key, properties: { hardware } }) => {
      if (hardware == null) return;
      const own = channels.filter((c) => c.device === key);
      reportConflicts(portClaims(own), "ctr", indexOf, report);
    });
  });

/**
 * Checks a digital config against the ports of its device: each line's port, its
 * place in the port's width, and the lines two channels share.
 */
export const digitalHardwareZ = (
  device: Device | undefined,
  direction: "input" | "output",
): z.ZodType =>
  check<{ channels: DigitalChannel[] }>(({ channels }, report) => {
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
    const indexOf = indexes(channels);
    lineClaims(channels).forEach((list) => {
      if (!conflicts(list)) return;
      list.forEach(({ channel: key, disabled }) => {
        if (disabled) return;
        const i = indexOf.get(key) as number;
        const { port, line } = channels[i];
        report(["channels", i, "line"], `Another channel uses port${port}/line${line}`);
      });
    });
  });
