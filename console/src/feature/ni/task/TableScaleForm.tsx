// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Form, Input, Select, state } from "@synnaxlabs/pluto";
import { binary, type record } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";
import { z } from "zod";

import { FS } from "@/platform/fs";

const tableSchema = z.record(z.string(), z.array(z.unknown()));

const COLUMNS = [{ name: "Pre-scaled" }, { name: "Scaled" }];

/** Keeps the appended row's pre-scaled value above the one before it. */
const createRow = (rows: number[][]): number[] => {
  const last = rows.at(-1);
  return [last == null ? 0 : last[0] + 1, 0];
};

/**
 * @returns The error to show when the two columns hold a different number of values, or
 * null when they pair up.
 */
const lengthMismatch = (preScaled: number[], scaled: number[]): string | null =>
  preScaled.length === scaled.length
    ? null
    : `Pre-scaled ${preScaled.length} values and scaled ${scaled.length} values must be the same length`;

export interface TableScaleFormProps {
  /** The form path of the table scale. */
  prefix: string;
}

/**
 * Edits the pre-scaled and scaled value pairs of a table scale. The pairs are typed in
 * directly or loaded from two columns of a CSV.
 */
export const TableScaleForm = ({ prefix }: TableScaleFormProps): ReactElement => {
  const [rawCol, setRawCol] = state.usePersisted<string>("Raw", `${prefix}.rawCol`);
  const [scaledCol, setScaledCol] = state.usePersisted<string>(
    "Scaled",
    `${prefix}.scaledCol`,
  );
  const [colOptions, setColOptions] = state.usePersisted<record.KeyedNamed<string>[]>(
    [],
    `${prefix}.colOptions`,
  );
  const [fileName, setFileName] = state.usePersisted<string>("", `${prefix}.path`);
  // The parsed table persists beside the form state, so a column change after a
  // remount recomputes the values without re-reading the file.
  const [table, setTable] = state.usePersisted<Record<string, unknown[]>>(
    {},
    `${prefix}.table`,
  );
  const preScaled = Form.useField<number[]>(`${prefix}.preScaledVals`);
  const scaled = Form.useField<number[]>(`${prefix}.scaledVals`);

  const rows = useMemo(() => {
    const length = Math.max(preScaled.value.length, scaled.value.length);
    return Array.from({ length }, (_, i) => [
      preScaled.value[i] ?? 0,
      scaled.value[i] ?? 0,
    ]);
  }, [preScaled.value, scaled.value]);

  const applyColumns = (
    value: Record<string, unknown[]>,
    raw: string,
    scaledKey: string,
  ) => {
    const preScaledValues = value[raw] as number[] | undefined;
    const scaledValues = value[scaledKey] as number[] | undefined;
    if (preScaledValues != null && scaledValues != null) {
      const message = lengthMismatch(preScaledValues, scaledValues);
      if (message != null) {
        // Loading the pair would pad the shorter column with zeros, giving the scale
        // points the CSV never held.
        preScaled.setStatus({ variant: "error", message });
        return;
      }
    }
    if (preScaledValues != null) preScaled.onChange(preScaledValues);
    if (scaledValues != null) scaled.onChange(scaledValues);
  };

  const handleFileChange = (value: z.infer<typeof tableSchema>, name: string) => {
    setFileName(name);
    setTable(value);
    const keys = Object.keys(value).filter(
      (key) =>
        Array.isArray(value[key]) && value[key].every((v) => isFinite(Number(v))),
    );
    setColOptions(keys.map((key) => ({ key, name: key })));
    const raw = keys.length > 0 ? keys[0] : rawCol;
    const scaledKey = keys.length > 1 ? keys[1] : scaledCol;
    if (keys.length > 0) setRawCol(raw);
    if (keys.length > 1) setScaledCol(scaledKey);
    applyColumns(value, raw, scaledKey);
  };

  const handleRawColChange = (value: string) => {
    setRawCol(value);
    applyColumns(table, value, scaledCol);
  };

  const handleScaledColChange = (value: string) => {
    setScaledCol(value);
    applyColumns(table, rawCol, value);
  };

  const handleRowsChange = (next: number[][]) => {
    preScaled.onChange(next.map(([raw]) => raw));
    scaled.onChange(next.map(([, value]) => value));
  };

  const { preview } = preScaled;
  const status =
    preScaled.status.variant !== "success" ? preScaled.status : scaled.status;
  // A scale stored with unpaired columns renders zeros for the values it never held, so
  // name the problem before an edit commits them.
  const mismatch = lengthMismatch(preScaled.value, scaled.value);
  return (
    <>
      {preview !== true && (
        <>
          <Input.Item label="Table CSV" padHelpText>
            <FS.InputFile<typeof tableSchema>
              value={fileName}
              onChange={handleFileChange}
              title="Select a table CSV"
              extension="csv"
              schema={tableSchema}
              decoder={binary.CSV_CODEC}
            />
          </Input.Item>
          {colOptions.length > 0 && (
            <Flex.Box x>
              <Input.Item label="Raw column" padHelpText grow>
                <Select.Static
                  resourceName="raw column"
                  value={rawCol}
                  onChange={handleRawColChange}
                  data={colOptions}
                />
              </Input.Item>
              <Input.Item label="Scaled column" padHelpText grow>
                <Select.Static
                  resourceName="scaled column"
                  value={scaledCol}
                  onChange={handleScaledColChange}
                  data={colOptions}
                />
              </Input.Item>
            </Flex.Box>
          )}
        </>
      )}
      <Input.Item
        label="Values"
        padHelpText
        helpText={mismatch ?? status.message}
        status={mismatch != null ? "error" : status.variant}
      >
        <Input.Table
          columns={COLUMNS}
          value={rows}
          onChange={handleRowsChange}
          createRow={createRow}
          preview={preview}
        />
      </Input.Item>
    </>
  );
};
