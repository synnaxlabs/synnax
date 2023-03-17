// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
//

import {
  convertRenderV,
  RenderableRecord,
  Bound,
  Direction,
  inBounds,
} from "@synnaxlabs/x";

export interface TableColumn<E extends RenderableRecord<E>> {
  key: keyof E;
  name?: string;
  width?: number;
}

export interface TableHighlight<E extends RenderableRecord<E>> {
  key: string;
  columns?: Array<keyof E>;
  rows?: Bound;
  color: string;
}

export interface TableProps<E extends RenderableRecord<E>> {
  columns: Array<TableColumn<E>>;
  data: E[];
  highlights?: Array<TableHighlight<E>>;
}

export const Table = <E extends RenderableRecord<E>>({
  columns,
  data,
  highlights = [],
}: TableProps<E>): JSX.Element => {
  return (
    <div style={{ overflow: "scroll" }}>
      <table style={{ overflow: "scroll" }}>
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
            <TableRow<E>
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
};

interface TableRowProps<E extends RenderableRecord<E>> {
  index: number;
  dataLength: number;
  columns: Array<TableColumn<E>>;
  data: E;
  highlights: Array<TableHighlight<E>>;
}

const TableRow = <E extends RenderableRecord<E>>({
  index,
  dataLength,
  columns,
  data,
  highlights,
}: TableRowProps<E>): JSX.Element => (
  <tr>
    {columns.map((col) => (
      <TableCell<E>
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

interface TableCellProps<E extends RenderableRecord<E>> {
  index: number;
  dataLength: number;
  highlights: Array<TableHighlight<E>>;
  data: E;
  column: TableColumn<E>;
}

const TableCell = <E extends RenderableRecord<E>>({
  index,
  dataLength,
  highlights,
  data,
  column,
}: TableCellProps<E>): JSX.Element => {
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
      />
    );
  }

  const left = highlights.filter(({ rows, columns, key }) => {
    const rowValid = rows != null ? inBounds(index, rows) : true;
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
      />
    );
  }

  const right = highlights.filter(({ rows, columns, key }) => {
    const rowValid = rows != null ? inBounds(index, rows) : true;
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
      />
    );
  }

  return (
    <td>
      {elements}
      {convertRenderV(data[column.key])}
    </td>
  );
};

const buildGradient = (
  colors: string[],
  direction: Direction,
  reverse: boolean
): string => {
  const count = colors.length;
  const gradient = colors.map((color, i) => {
    const start = (i * 100) / count;
    const end = ((i + 1) * 100) / count;
    return `${color} ${start}% ${end}%`;
  });
  let dir;
  switch (direction) {
    case "x":
      dir = reverse ? "to right" : "to left";
      break;
    case "y":
      dir = reverse ? "to top" : "to bottom";
      break;
  }
  return `linear-gradient(${dir}, ${gradient.join(", ")})`;
};
