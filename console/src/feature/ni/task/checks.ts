// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ni } from "@synnaxlabs/client";
import { type bounds, unique } from "@synnaxlabs/x";
import type { z } from "zod";

import type { Device } from "@/feature/ni/device/types";
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

const formatMinRate = (hz: number): string => {
  const rate = hz.toLocaleString(undefined, {
    maximumSignificantDigits: 6,
    roundingMode: "ceil",
  });
  return `${rate} Hz`;
};

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
    const min = hardware.aiMinRate;
    if (min !== 0 && config.sampleRate < min)
      return report(["sampleRate"], `${model} samples at least ${formatMinRate(min)}`);
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

interface RangedChannel {
  minVal: number;
  maxVal: number;
  customScale: ni.Scale;
}

interface Measure {
  /** The units the device ranges are in. */
  units: ni.Units;
  symbol: string;
}

const VOLTS: Measure = { units: "Volts", symbol: "V" };
const AMPS: Measure = { units: "Amps", symbol: "A" };

const evaluate = (coeffs: number[], x: number): number =>
  coeffs.reduceRight((sum, c) => sum * x + c, 0);

/**
 * Maps a scaled value back to the pre-scaled value that DAQmx checks against the
 * device ranges. Undefined when the scale gives no single pre-scaled value.
 */
const preScaled = (scale: ni.Scale, value: number): number | undefined => {
  switch (scale.type) {
    case "none":
      return value;
    case "linear":
      if (scale.slope === 0) return undefined;
      return (value - scale.yIntercept) / scale.slope;
    case "map": {
      const { preScaledMin, preScaledMax, scaledMin, scaledMax } = scale;
      if (scaledMin === scaledMax) return undefined;
      const fraction = (value - scaledMin) / (scaledMax - scaledMin);
      return preScaledMin + fraction * (preScaledMax - preScaledMin);
    }
    case "polynomial":
      if (scale.reverseCoeffs.length === 0) return undefined;
      return evaluate(scale.reverseCoeffs, value);
    case "table":
      return undefined;
  }
};

const format = (value: number): number => Number(value.toPrecision(6));

/**
 * Reports a channel whose expected values fit in no range of the device. An empty list
 * means DAQmx did not report the ranges. With a custom scale, the values are in scaled
 * units, so the check maps them back to pre-scaled units first.
 */
const checkRange = (
  { minVal, maxVal, customScale }: RangedChannel,
  ranges: bounds.Bounds[],
  { units, symbol }: Measure,
  path: (string | number)[],
  report: Task.ReportIssue,
): void => {
  if (ranges.length === 0) return;
  if (customScale.type !== "none" && customScale.preScaledUnits !== units) return;
  // Scale math can land one rounding error past a bound the user meant to hit.
  const toDevice = (value: number): number | undefined => {
    const result = preScaled(customScale, value);
    return result == null || customScale.type === "none" ? result : format(result);
  };
  const min = toDevice(minVal);
  const max = toDevice(maxVal);
  if (min == null || max == null) return;
  const lo = Math.min(min, max);
  const hi = Math.max(min, max);
  if (ranges.some(({ lower, upper }) => lower <= lo && hi <= upper)) return;
  const maxFits = ranges.some(({ lower, upper }) => lower <= max && max <= upper);
  const range = `${format(min)} to ${format(max)} ${symbol}`;
  let message = `The device has no range from ${range}`;
  if (customScale.type !== "none") {
    const given = `${minVal} to ${maxVal} ${customScale.scaledUnits}`;
    message += `, which the scale gives for ${given}`;
  }
  report([...path, maxFits ? "minVal" : "maxVal"], message);
};

/**
 * Checks an analog read config against the ports of its devices: the ports two
 * channels on a device share, each channel's port, type, terminal configuration, and
 * range, and the sample rate. Only the port checks run on a device with no port data.
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
      if (c.type === "ai_voltage")
        checkRange(c, hardware.aiVoltageRanges, VOLTS, ["channels", i], report);
      if (c.type === "ai_current" && c.shuntResistorLoc !== "External")
        checkRange(c, hardware.aiCurrentRanges, AMPS, ["channels", i], report);
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
      if (c.type === "ao_voltage")
        checkRange(c, hardware.aoVoltageRanges, VOLTS, ["channels", i], report);
      if (c.type === "ao_current")
        checkRange(c, hardware.aoCurrentRanges, AMPS, ["channels", i], report);
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
