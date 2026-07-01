// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/component/table/Table.css";

import { table } from "@synnaxlabs/client";
import {
  Access,
  Breadcrumb,
  Color,
  Flex,
  Form,
  Icon,
  Input,
  Select,
  Table,
  Text,
  Theming,
} from "@synnaxlabs/pluto";
import { color, deep, record, type text } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";

import { Cluster } from "@/component/cluster";
import { CSS } from "@/component/css";
import { Empty } from "@/component/empty";
import { Export } from "@/component/export";
import { Toolbar as Base } from "@/component/toolbar";
import { useExport } from "@/service/table/export";
import { Session } from "@/session";

const Internal = (): ReactElement => {
  const key = Table.useKey();
  const handleExport = useExport();
  const name = Table.useSelectName();
  const editable = Session.Table.useSelectEditable();
  const selectedCellKeys = Session.Table.useSelectSelectedCellKeys();
  const cellsByKey = Table.useSelectCells({ cellKeys: selectedCellKeys });
  const liveCellCount = cellsByKey.size;
  const singleSelectedKey =
    liveCellCount === 1 ? (cellsByKey.keys().next().value ?? null) : null;
  const selectedCellPos = Table.useCellPosition({ cellKey: singleSelectedKey ?? "" });
  return (
    <Base.Content>
      <Base.Header>
        <Flex.Box x align="center">
          <Breadcrumb.Breadcrumb>
            <Breadcrumb.Segment weight={500} color={10} level="h5">
              <Icon.Table />
              {name}
            </Breadcrumb.Segment>
            {selectedCellPos != null && (
              <Breadcrumb.Segment color={8}>
                {Table.getCellColumn(selectedCellPos.x)}
                {selectedCellPos.y + 1}
              </Breadcrumb.Segment>
            )}
            {liveCellCount > 1 && (
              <Breadcrumb.Segment color={8}>{liveCellCount} cells</Breadcrumb.Segment>
            )}
          </Breadcrumb.Breadcrumb>
        </Flex.Box>
        <Flex.Box x className={CSS.BE("table", "toolbar-buttons")} empty>
          <Export.ToolbarButton onExport={() => handleExport(key)} />
          <Cluster.CopyLinkToolbarButton
            name={name}
            ontologyID={table.ontologyID(key)}
          />
        </Flex.Box>
      </Base.Header>
      <Flex.Box full>
        {!editable ? (
          <NotEditableContent name={name} />
        ) : liveCellCount === 0 ? (
          <EmptyContent />
        ) : singleSelectedKey != null ? (
          <CellForm key={singleSelectedKey} cellKey={singleSelectedKey} />
        ) : (
          <MultiCellForm cellKeys={selectedCellKeys} />
        )}
      </Flex.Box>
    </Base.Content>
  );
};

export interface ToolbarProps {
  layoutKey: string;
}

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement => (
  <Table.Suspended tableKey={layoutKey}>
    <Internal />
  </Table.Suspended>
);

// buildVariantSwapActions returns one setCell action per cell whose variant
// differs from the target. Compatible fields survive the swap.
const buildVariantSwapActions = (
  cells: Iterable<[string, Table.Cell.Config]>,
  variant: Table.Cell.Variant,
  theme: Theming.Theme,
): table.Action[] => {
  const spec = Table.Cell.REGISTRY[variant];
  const actions: table.Action[] = [];
  for (const [key, cell] of cells) {
    if (cell.variant === variant) continue;
    const props = deep.overrideValidItems(
      cell.props,
      spec.defaultProps(theme),
      spec.schema,
    );
    actions.push(table.setCell({ cell: { key, variant, props } }));
  }
  return actions;
};

interface CellFormProps {
  cellKey: string;
}

const CellForm = ({ cellKey }: CellFormProps): ReactElement | null => {
  const cell = Table.useSelectCell({ cellKey });
  const dispatch = Table.useSingleDispatch();
  const theme = Theming.use();

  const handleVariantChange = useCallback(
    (variant: Table.Cell.Variant) => {
      if (cell != null)
        dispatch(buildVariantSwapActions([[cellKey, cell]], variant, theme));
    },
    [cell, cellKey, dispatch, theme],
  );

  const handleChange = useCallback(
    ({ values }: Form.OnChangeArgs<ReturnType<typeof record.unknownZ>>) => {
      if (cell == null) return;
      dispatch([
        table.setCell({
          cell: { key: cellKey, variant: cell.variant, props: deep.copy(values) },
        }),
      ]);
    },
    [cell, cellKey, dispatch],
  );

  const methods = Form.use<ReturnType<typeof record.unknownZ>>({
    values: cell != null ? deep.copy(cell.props) : {},
    schema: record.unknownZ(),
    onChange: handleChange,
    sync: true,
  });

  if (cell == null) return null;
  const C = Table.Cell.REGISTRY[cell.variant];
  return (
    <Form.Form<ReturnType<typeof record.unknownZ>> {...methods}>
      <C.Form onVariantChange={handleVariantChange} />
    </Form.Form>
  );
};

const EmptyContent = (): ReactElement => (
  <Text.Text status="disabled" center>
    No cell selected. Select a cell to view its properties.
  </Text.Text>
);

interface NotEditableContentProps {
  name: string;
}

const NotEditableContent = ({ name }: NotEditableContentProps): ReactElement => {
  const key = Table.useKey();
  const dispatch = Session.useDispatch();
  const hasUpdatePermission = Access.useUpdateGranted(table.ontologyID(key));
  return (
    <Empty.Action
      x
      message={`${name} is not editable.${hasUpdatePermission ? " To make changes," : ""}`}
      action={hasUpdatePermission ? "enable editing." : undefined}
      onClick={() => dispatch(Session.Table.setEditable({ key, editable: true }))}
    />
  );
};

const readCellColor = (cell: Table.Cell.Config): color.Hex | null => {
  switch (cell.variant) {
    case "text":
      return cell.props.backgroundColor == null
        ? null
        : color.hex(cell.props.backgroundColor);
    case "value":
      return color.hex(cell.props.color);
  }
};

const cellColorPatch = (
  cell: Table.Cell.Config,
  next: color.Color,
): Partial<record.Unknown> => {
  switch (cell.variant) {
    case "text":
      return { backgroundColor: next };
    case "value":
      return { color: next };
  }
};

interface MultiCellFormProps {
  cellKeys: string[];
}

const MultiCellForm = ({ cellKeys }: MultiCellFormProps): ReactElement => {
  const cellsByKey = Table.useSelectCells({ cellKeys });
  const dispatch = Table.useSingleDispatch();
  const theme = Theming.use();

  // Cells absent from the store are skipped (selection may include keys from
  // a removed row). One dispatch per call so undo collapses to one step.
  const applyPropPatch = useCallback(
    (
      keys: string[],
      patch: (cell: Table.Cell.Config) => Partial<record.Unknown> | null,
    ) => {
      const actions: table.Action[] = [];
      for (const key of keys) {
        const cell = cellsByKey.get(key);
        if (cell == null) continue;
        const next = patch(cell);
        if (next == null) continue;
        actions.push(
          table.setCell({
            cell: {
              key,
              variant: cell.variant,
              props: { ...cell.props, ...next },
            },
          }),
        );
      }
      dispatch(actions);
    },
    [cellsByKey, dispatch],
  );

  const variants = useMemo(() => {
    const s = new Set<Table.Cell.Variant>();
    cellsByKey.forEach((c) => s.add(c.variant));
    return s;
  }, [cellsByKey]);
  const commonVariant =
    variants.size === 1 ? (variants.values().next().value ?? null) : null;

  const handleVariantChange = useCallback(
    (variant: Table.Cell.Variant) => {
      dispatch(buildVariantSwapActions(cellsByKey, variant, theme));
    },
    [cellsByKey, dispatch, theme],
  );

  const colorGroups = useMemo(() => {
    const groups = new Map<color.Hex, string[]>();
    cellsByKey.forEach((cell, key) => {
      const hex = readCellColor(cell);
      if (hex == null) return;
      const existing = groups.get(hex);
      if (existing != null) existing.push(key);
      else groups.set(hex, [key]);
    });
    return groups;
  }, [cellsByKey]);

  const handleColorChange = useCallback(
    (groupKeys: string[], next: color.Color) =>
      applyPropPatch(groupKeys, (cell) => cellColorPatch(cell, next)),
    [applyPropPatch],
  );

  const commonLevel = useMemo((): text.Level | undefined => {
    let result: text.Level | undefined;
    for (const cell of cellsByKey.values())
      if (result == null) result = cell.props.level;
      else if (result !== cell.props.level) return undefined;

    return result;
  }, [cellsByKey]);

  const handleLevelChange = useCallback(
    (level: text.Level) => applyPropPatch(cellKeys, () => ({ level })),
    [applyPropPatch, cellKeys],
  );

  return (
    <Flex.Box
      x
      align="start"
      gap="large"
      className={CSS.BE("table", "multi-cell-form")}
    >
      <Input.Item label="Variant" padHelpText={false}>
        <Table.Cell.SelectVariant
          value={commonVariant ?? undefined}
          onChange={handleVariantChange}
        />
      </Input.Item>
      {colorGroups.size > 0 && (
        <Input.Item label="Selection colors" align="start" padHelpText={false}>
          <Flex.Box x>
            {Array.from(colorGroups.entries()).map(([hex, keys]) => (
              <Color.Swatch
                key={keys[0]}
                value={hex}
                onChange={(c: color.Color) => handleColorChange(keys, c)}
              />
            ))}
          </Flex.Box>
        </Input.Item>
      )}
      <Input.Item label="Size" padHelpText={false}>
        <Select.Text.Level value={commonLevel} onChange={handleLevelChange} />
      </Input.Item>
    </Flex.Box>
  );
};
