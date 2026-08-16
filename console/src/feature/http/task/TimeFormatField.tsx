// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component, Form as PForm, Select } from "@synnaxlabs/pluto";
import { type FC } from "react";

import { type TimeFormat } from "@/feature/http/task/types";
import { CSS } from "@/platform/css";

const DATA: Select.StaticEntry<TimeFormat>[] = [
  { key: "iso8601", name: "ISO 8601" },
  { key: "unix_sec", name: "Unix (s)" },
  { key: "unix_ms", name: "Unix (ms)" },
  { key: "unix_us", name: "Unix (µs)" },
  { key: "unix_ns", name: "Unix (ns)" },
];

const renderSelect = Component.renderProp(
  (
    p: Omit<
      Select.StaticProps<TimeFormat, Select.StaticEntry<TimeFormat>>,
      "data" | "resourceName"
    >,
  ) => (
    <Select.Static<TimeFormat, Select.StaticEntry<TimeFormat>>
      {...p}
      data={DATA}
      resourceName="time format"
    />
  ),
);

export interface TimeFormatFieldProps {
  path: string;
  label: string;
}

/**
 * Selects the encoding of a timestamp carried as a JSON value. The caller decides when
 * a timestamp format applies; mounting seeds `path` with ISO 8601 so the choice is
 * visible and stored instead of silently absent.
 */
export const TimeFormatField: FC<TimeFormatFieldProps> = ({ path, label }) => (
  <PForm.Field<TimeFormat>
    path={path}
    label={label}
    defaultValue="iso8601"
    className={CSS.B("time-format")}
  >
    {renderSelect}
  </PForm.Field>
);
