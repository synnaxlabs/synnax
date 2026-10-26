// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  MdAcUnit,
  MdAir,
  MdApartment,
  MdBarChart,
  MdBiotech,
  MdBuild,
  MdCable,
  MdChecklist,
  MdDangerous,
  MdDashboard,
  MdFactory,
  MdFlag,
  MdFlight,
  MdGpsFixed,
  MdGraphicEq,
  MdGroups,
  MdLightbulb,
  MdLocalFireDepartment,
  MdLock,
  MdMap,
  MdMedicalServices,
  MdMonitorHeart,
  MdNotifications,
  MdOilBarrel,
  MdPieChart,
  MdPower,
  MdPrecisionManufacturing,
  MdPropane,
  MdPublic,
  MdRocketLaunch,
  MdSatelliteAlt,
  MdScience,
  MdSensors,
  MdSettings,
  MdStar,
  MdStorage,
  MdTableChart,
  MdTerminal,
  MdTimer,
  MdVerifiedUser,
  MdVisibility,
  MdWarning,
  MdWaterDrop,
  MdWbSunny,
} from "react-icons/md";

import { Device } from "@/icon/Device";
import { type FC, wrapSVGIcon } from "@/icon/Icon";
import { Pump } from "@/icon/Pump";
import { Rack } from "@/icon/Rack";
import { Arc, LinePlot, Range, Schematic, Task } from "@/icon/registry";
import { Tank } from "@/icon/Tank";
import { Units } from "@/icon/Units";
import { Valve } from "@/icon/Valve";

/** An icon a user can pick for an object, such as a panel. */
export interface Pickable {
  /** Identifies the icon in stored data. A shipped key never changes or goes away. */
  key: string;
  /** Labels the icon in tooltips and for screen readers. */
  name: string;
  /** Extra search terms, such as hardware synonyms. */
  tags: string[];
  icon: FC;
}

/** The icons a user can pick, in display order. */
export const PICKABLE: readonly Pickable[] = [
  {
    key: "line-plot",
    name: "Line plot",
    tags: ["chart", "graph", "trend", "time series"],
    icon: LinePlot,
  },
  {
    key: "schematic",
    name: "Schematic",
    tags: ["diagram", "P&ID", "piping", "process", "synoptic"],
    icon: Schematic,
  },
  {
    key: "task",
    name: "Task",
    tags: ["acquisition", "daq", "read", "write", "driver"],
    icon: Task,
  },
  {
    key: "arc",
    name: "Arc",
    tags: ["automation", "sequence", "control", "script", "program"],
    icon: Arc,
  },
  {
    key: "device",
    name: "Device",
    tags: ["hardware", "module", "daq", "chip"],
    icon: Device,
  },
  {
    key: "range",
    name: "Range",
    tags: ["test", "run", "time range", "event", "period"],
    icon: Range,
  },
  {
    key: "bar-chart",
    name: "Bar chart",
    tags: ["histogram", "graph", "statistics"],
    icon: wrapSVGIcon(MdBarChart, "bar-chart"),
  },
  {
    key: "pie-chart",
    name: "Pie chart",
    tags: ["graph", "share", "breakdown"],
    icon: wrapSVGIcon(MdPieChart, "pie-chart"),
  },
  {
    key: "table",
    name: "Table",
    tags: ["grid", "spreadsheet", "values", "readings"],
    icon: wrapSVGIcon(MdTableChart, "table"),
  },
  {
    key: "terminal",
    name: "Terminal",
    tags: ["console", "log", "command", "script"],
    icon: wrapSVGIcon(MdTerminal, "terminal"),
  },
  {
    key: "waveform",
    name: "Waveform",
    tags: ["signal", "vibration", "audio", "frequency", "spectrum"],
    icon: wrapSVGIcon(MdGraphicEq, "waveform"),
  },
  {
    key: "pulse",
    name: "Pulse",
    tags: ["heartbeat", "activity", "health", "monitor"],
    icon: wrapSVGIcon(MdMonitorHeart, "pulse"),
  },
  {
    key: "dashboard",
    name: "Dashboard",
    tags: ["overview", "layout", "tiles", "panels"],
    icon: wrapSVGIcon(MdDashboard, "dashboard"),
  },
  {
    key: "map",
    name: "Map",
    tags: ["location", "site", "layout", "field"],
    icon: wrapSVGIcon(MdMap, "map"),
  },
  {
    key: "database",
    name: "Database",
    tags: ["storage", "data", "archive", "server"],
    icon: wrapSVGIcon(MdStorage, "database"),
  },
  {
    key: "valve",
    name: "Valve",
    tags: ["solenoid", "actuator", "flow", "fluid"],
    icon: Valve,
  },
  {
    key: "pump",
    name: "Pump",
    tags: ["flow", "fluid", "compressor", "turbopump"],
    icon: Pump,
  },
  {
    key: "tank",
    name: "Tank",
    tags: ["vessel", "reservoir", "storage", "propellant"],
    icon: Tank,
  },
  {
    key: "barrel",
    name: "Barrel",
    tags: ["oil", "drum", "fuel", "storage"],
    icon: wrapSVGIcon(MdOilBarrel, "barrel"),
  },
  {
    key: "gas-cylinder",
    name: "Gas cylinder",
    tags: ["gas", "nitrogen", "helium", "propane", "bottle"],
    icon: wrapSVGIcon(MdPropane, "gas-cylinder"),
  },
  {
    key: "air",
    name: "Air",
    tags: ["wind", "flow", "ventilation", "hvac", "purge"],
    icon: wrapSVGIcon(MdAir, "air"),
  },
  {
    key: "rack",
    name: "Rack",
    tags: ["driver", "server", "chassis", "host"],
    icon: Rack,
  },
  {
    key: "power",
    name: "Power",
    tags: ["plug", "electrical", "outlet", "supply"],
    icon: wrapSVGIcon(MdPower, "power"),
  },
  {
    key: "cable",
    name: "Cable",
    tags: ["wiring", "connector", "harness"],
    icon: wrapSVGIcon(MdCable, "cable"),
  },
  {
    key: "pressure",
    name: "Pressure",
    tags: ["psi", "bar", "transducer", "gauge"],
    icon: Units.Pressure,
  },
  {
    key: "temperature",
    name: "Temperature",
    tags: ["thermocouple", "rtd", "thermometer", "heat"],
    icon: Units.Temperature,
  },
  {
    key: "force",
    name: "Force",
    tags: ["load cell", "newton", "weight", "thrust"],
    icon: Units.Force,
  },
  {
    key: "torque",
    name: "Torque",
    tags: ["moment", "rotation"],
    icon: Units.Torque,
  },
  {
    key: "strain",
    name: "Strain",
    tags: ["strain gauge", "bridge", "deformation", "stress"],
    icon: Units.Strain,
  },
  {
    key: "acceleration",
    name: "Acceleration",
    tags: ["accelerometer", "vibration", "imu"],
    icon: Units.Acceleration,
  },
  {
    key: "velocity",
    name: "Velocity",
    tags: ["speed", "flow rate", "rate"],
    icon: Units.Velocity,
  },
  {
    key: "voltage",
    name: "Voltage",
    tags: ["volts", "electrical", "bolt"],
    icon: Units.Voltage,
  },
  {
    key: "current",
    name: "Current",
    tags: ["amps", "ampere", "electrical"],
    icon: Units.Current,
  },
  {
    key: "resistance",
    name: "Resistance",
    tags: ["ohms", "resistor"],
    icon: Units.Resistance,
  },
  {
    key: "rocket",
    name: "Rocket",
    tags: ["launch", "engine", "propulsion", "flight"],
    icon: wrapSVGIcon(MdRocketLaunch, "rocket"),
  },
  {
    key: "satellite",
    name: "Satellite",
    tags: ["space", "orbit", "spacecraft", "communication"],
    icon: wrapSVGIcon(MdSatelliteAlt, "satellite"),
  },
  {
    key: "airplane",
    name: "Airplane",
    tags: ["aircraft", "flight", "aviation"],
    icon: wrapSVGIcon(MdFlight, "airplane"),
  },
  {
    key: "flame",
    name: "Flame",
    tags: ["fire", "combustion", "ignition", "hot fire"],
    icon: wrapSVGIcon(MdLocalFireDepartment, "flame"),
  },
  {
    key: "flask",
    name: "Flask",
    tags: ["lab", "chemistry", "science", "experiment"],
    icon: wrapSVGIcon(MdScience, "flask"),
  },
  {
    key: "microscope",
    name: "Microscope",
    tags: ["lab", "inspection", "science"],
    icon: wrapSVGIcon(MdBiotech, "microscope"),
  },
  {
    key: "robot-arm",
    name: "Robot arm",
    tags: ["automation", "manufacturing", "robot", "actuator"],
    icon: wrapSVGIcon(MdPrecisionManufacturing, "robot-arm"),
  },
  {
    key: "wrench",
    name: "Wrench",
    tags: ["tool", "maintenance", "repair", "setup"],
    icon: wrapSVGIcon(MdBuild, "wrench"),
  },
  {
    key: "gear",
    name: "Gear",
    tags: ["settings", "mechanical", "configuration", "machine"],
    icon: wrapSVGIcon(MdSettings, "gear"),
  },
  {
    key: "timer",
    name: "Timer",
    tags: ["stopwatch", "countdown", "sequence", "duration"],
    icon: wrapSVGIcon(MdTimer, "timer"),
  },
  {
    key: "warning",
    name: "Warning",
    tags: ["caution", "alert", "alarm", "hazard"],
    icon: wrapSVGIcon(MdWarning, "warning"),
  },
  {
    key: "danger",
    name: "Danger",
    tags: ["abort", "stop", "critical", "emergency"],
    icon: wrapSVGIcon(MdDangerous, "danger"),
  },
  {
    key: "bell",
    name: "Bell",
    tags: ["alarm", "notification", "alert"],
    icon: wrapSVGIcon(MdNotifications, "bell"),
  },
  {
    key: "shield",
    name: "Shield",
    tags: ["safety", "protection", "interlock", "secure"],
    icon: wrapSVGIcon(MdVerifiedUser, "shield"),
  },
  {
    key: "lock",
    name: "Lock",
    tags: ["locked", "access", "permission", "secure"],
    icon: wrapSVGIcon(MdLock, "lock"),
  },
  {
    key: "first-aid",
    name: "First aid",
    tags: ["medical", "safety", "health", "emergency"],
    icon: wrapSVGIcon(MdMedicalServices, "first-aid"),
  },
  {
    key: "checklist",
    name: "Checklist",
    tags: ["procedure", "steps", "tasks", "checks"],
    icon: wrapSVGIcon(MdChecklist, "checklist"),
  },
  {
    key: "flag",
    name: "Flag",
    tags: ["milestone", "marker", "finish"],
    icon: wrapSVGIcon(MdFlag, "flag"),
  },
  {
    key: "target",
    name: "Target",
    tags: ["aim", "objective", "setpoint", "crosshair"],
    icon: wrapSVGIcon(MdGpsFixed, "target"),
  },
  {
    key: "eye",
    name: "Eye",
    tags: ["view", "watch", "visibility", "monitor"],
    icon: wrapSVGIcon(MdVisibility, "eye"),
  },
  {
    key: "factory",
    name: "Factory",
    tags: ["plant", "manufacturing", "production", "facility"],
    icon: wrapSVGIcon(MdFactory, "factory"),
  },
  {
    key: "building",
    name: "Building",
    tags: ["office", "site", "facility", "lab"],
    icon: wrapSVGIcon(MdApartment, "building"),
  },
  {
    key: "globe",
    name: "Globe",
    tags: ["world", "network", "remote", "global"],
    icon: wrapSVGIcon(MdPublic, "globe"),
  },
  {
    key: "people",
    name: "People",
    tags: ["team", "group", "users", "crew"],
    icon: wrapSVGIcon(MdGroups, "people"),
  },
  {
    key: "sensor",
    name: "Sensor",
    tags: ["telemetry", "wireless", "signal", "broadcast"],
    icon: wrapSVGIcon(MdSensors, "sensor"),
  },
  {
    key: "drop",
    name: "Drop",
    tags: ["water", "liquid", "fluid", "coolant"],
    icon: wrapSVGIcon(MdWaterDrop, "drop"),
  },
  {
    key: "snowflake",
    name: "Snowflake",
    tags: ["cold", "cryogenic", "freeze", "cooling"],
    icon: wrapSVGIcon(MdAcUnit, "snowflake"),
  },
  {
    key: "sun",
    name: "Sun",
    tags: ["heat", "solar", "light", "day"],
    icon: wrapSVGIcon(MdWbSunny, "sun"),
  },
  {
    key: "lightbulb",
    name: "Lightbulb",
    tags: ["idea", "light", "lamp"],
    icon: wrapSVGIcon(MdLightbulb, "lightbulb"),
  },
  {
    key: "star",
    name: "Star",
    tags: ["favorite", "important", "highlight"],
    icon: wrapSVGIcon(MdStar, "star"),
  },
];

/** @returns the pickable icon with the key, or undefined when no icon has it. */
export const getPickable = (key: string): Pickable | undefined =>
  PICKABLE.find((p) => p.key === key);
