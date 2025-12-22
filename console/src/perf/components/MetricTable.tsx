// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/perf/components/MetricTable.css";

import { Text } from "@synnaxlabs/pluto";
import { memo,type ReactElement } from "react";

import { DISPLAY_LIMIT, RESIZE_THRESHOLD } from "@/perf/constants";

/** Data structure returned by collectors with truncation metadata */
export interface MetricTableData<T> {
  data: T[];
  total: number;
  truncated: boolean;
}

/** Column definition for metric table display */
export interface MetricTableColumn<T> {
  getValue: (item: T, index: number) => string | number;
  color?: number;
}

/** Props for the data table component */
interface DataTableProps<T> {
  data: T[];
  columns: MetricTableColumn<T>[];
  getKey: (item: T, index: number) => string;
  getTooltip?: (item: T) => string | undefined;
}

function DataTableImpl<T>({
  data,
  columns,
  getKey,
  getTooltip,
}: DataTableProps<T>): ReactElement {
  const isResizable = data.length >= RESIZE_THRESHOLD;
  return (
    <div className="console-perf-data-table-wrapper" data-resizable={isResizable}>
      <table className="console-perf-data-table">
        <tbody>
          {data.map((item, index) => {
            const key = getKey(item, index);
            const tooltip = getTooltip?.(item);
            return (
              <tr key={key} title={tooltip}>
                {columns.map((col, colIndex) => {
                  const textProps: any = { level: "small" };
                  if (col.color != null) textProps.color = col.color;
                  return (
                    <td key={`${key}-${colIndex}`}>
                      <Text.Text {...textProps}>
                        {col.getValue(item, index)}
                      </Text.Text>
                    </td>
                  );
                })}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

const DataTable = memo(DataTableImpl) as typeof DataTableImpl;

/** Props for metric table component */
export interface MetricTableProps<T> {
  result: MetricTableData<T>;
  columns: MetricTableColumn<T>[];
  getKey: (item: T, index: number) => string;
  getTooltip?: (item: T) => string | undefined;
}

function MetricTableImpl<T>({
  result,
  columns,
  getKey,
  getTooltip,
}: MetricTableProps<T>): ReactElement {
  const displayData = result.data.slice(0, DISPLAY_LIMIT);
  const isTruncated = result.data.length > DISPLAY_LIMIT;

  return (
    <>
      <DataTable
        data={displayData}
        columns={columns}
        getKey={getKey}
        getTooltip={getTooltip}
      />
      {isTruncated && (
        <Text.Text level="small" className="console-perf-truncation-text">
          Showing {displayData.length} of {result.total}
        </Text.Text>
      )}
    </>
  );
}

export const MetricTable = memo(MetricTableImpl) as typeof MetricTableImpl;
