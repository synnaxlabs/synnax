// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  bounds,
  convertRenderV,
  type direction,
  type Key,
  type Keyed,
} from "@synnaxlabs/x";
import { type ReactElement } from "react";

export interface TableColumn<K extends Key, E extends Keyed<K>> {
  key: keyof E;
  name?: string;
  width?: number;
  type?: "code" | "html";
}

export interface TableHighlight<K extends Key, E extends Keyed<K>> {
  key: string;
  columns?: Array<keyof E>;
  rows?: bounds.Bounds;
  color: string;
}

export interface TableProps<K extends Key, E extends Keyed<K>> {
  columns: Array<TableColumn<K, E>>;
  data: E[];
  highlights?: Array<TableHighlight<K, E>>;
}

export const Table = <K extends Key, E extends Keyed<K>>({
  columns,
  data,
  highlights = [],
}: TableProps<Key, E>): ReactElement => (
  <div style={{ overflowX: "auto", paddingLeft: 2 }}>
    <table>
      <thead>
        <tr>
          {columns.map(({ key, name, width }) => (
            <th key={key as string} style={{ width }}>
              {name ?? (key as string)}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {data.map((row, i) => (
          <TableRow<K, E>
            key={i}
            columns={columns}
            data={row}
            highlights={highlights}
            index={i}
            dataLength={data.length}
          />
        ))}
      </tbody>
    </table>
  </div>
);

interface TableRowProps<K extends Key, E extends Keyed<K>> {
  index: number;
  dataLength: number;
  columns: Array<TableColumn<K, E>>;
  data: E;
  highlights: Array<TableHighlight<K, E>>;
}

const TableRow = <K extends Key, E extends Keyed<K>>({
  index,
  dataLength,
  columns,
  data,
  highlights,
}: TableRowProps<K, E>): ReactElement => (
  <tr>
    {columns.map((col) => (
      <TableCell<K, E>
        key={col.key as string}
        index={index}
        dataLength={dataLength}
        highlights={highlights}
        data={data}
        column={col}
      />
    ))}
  </tr>
);

interface TableCellProps<K extends Key, E extends Keyed<K>> {
  index: number;
  dataLength: number;
  highlights: Array<TableHighlight<K, E>>;
  data: E;
  column: TableColumn<K, E>;
}

const TableCell = <K extends Key, E extends Keyed<K>>({
  index,
  dataLength,
  highlights,
  data,
  column,
}: TableCellProps<K, E>): ReactElement | null => {
  const endings = highlights.filter(({ rows, columns }) => {
    const rowValid = rows != null ? rows.upper === index : index === dataLength - 1;
    const colValid = columns != null ? columns.includes(column.key) : true;
    return rowValid && colValid;
  });

  const startings = highlights.filter(({ rows, columns }) => {
    const rowValid = rows != null ? rows.lower === index : index === 0;
    const colValid = columns != null ? columns.includes(column.key) : true;
    return rowValid && colValid;
  });

  const upperColors = [...endings, ...startings].map(({ color }) => color);

  const elements = [];
  if (upperColors.length > 0) {
    const background = buildGradient(upperColors, "y", false);
    elements.push(
      <div
        style={{
          height: upperColors.length * 2,
          width: "calc(100% + 2px)",
          background,
          position: "absolute",
          top: -upperColors.length,
          left: -1,
        }}
      />,
    );
  }

  const left = highlights.filter(({ rows, columns, key }) => {
    const rowValid =
      rows != null ? bounds.contains(bounds.construct(rows), index) : true;
    const colValid = columns != null ? columns[0] === column.key : true;
    const isEnd = endings.some(({ key: pKey }) => key === pKey);
    return rowValid && colValid && !isEnd;
  });

  const leftColors = left.map(({ color }) => color);

  if (leftColors.length > 0) {
    const background = buildGradient(leftColors, "x", false);
    elements.push(
      <div
        style={{
          height: "calc(100% + 2px)",
          width: leftColors.length * 2,
          background,
          position: "absolute",
          top: -1,
          left: -leftColors.length,
        }}
      />,
    );
  }

  const right = highlights.filter(({ rows, columns, key }) => {
    const rowValid = rows != null ? bounds.contains(rows, index) : true;
    const colValid =
      columns != null ? columns[columns.length - 1] === column.key : true;
    const isEnd = endings.some(({ key: pKey }) => key === pKey);
    return rowValid && colValid && !isEnd;
  });

  const rightColors = right.map(({ color }) => color);

  if (rightColors.length > 0) {
    const background = buildGradient(rightColors, "x", true);
    elements.push(
      <div
        style={{
          height: "calc(100% + 2px)",
          width: rightColors.length * 2,
          background,
          position: "absolute",
          top: -1,
          right: -rightColors.length,
        }}
      />,
    );
  }

  let content: ReactElement | string | number | undefined = convertRenderV(
    data[column.key],
  );
  if (column.type === "code") content = <code>{content}</code>;
  if (column.type === "html")
    content = <div dangerouslySetInnerHTML={{ __html: content as string }} />;

  return (
    <td>
      {elements}
      {content}
    </td>
  );
};

const buildGradient = (
  colors: string[],
  direction: direction.Direction,
  reverse: boolean,
): string => {
  const count = colors.length;
  const gradient = colors.map((color, i) => {
    const start = (i * 100) / count;
    const end = ((i + 1) * 100) / count;
    return `${color} ${start}% ${end}%`;
  });
  let dir;
  if (direction === "x") dir = reverse ? "to right" : "to left";
  else dir = reverse ? "to top" : "to bottom";
  return `linear-gradient(${dir}, ${gradient.join(", ")})`;
};
