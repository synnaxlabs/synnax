// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
  Table as Base,
  TableCells,
  Text,
} from "@synnaxlabs/pluto";
import { color, deep, record, type text } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch, useStore } from "react-redux";

import { Cluster } from "@/cluster";
import { EmptyAction, Toolbar as Tb } from "@/components";
import { Export } from "@/export";
import { Layout } from "@/layout";
import { type RootState } from "@/store";
import { useExport } from "@/table/export";
import {
  useSelectEditable,
  useSelectPendingUpload,
  useSelectSelectedCellKeys,
} from "@/table/selectors";
import { setEditable } from "@/table/slice";

export interface ToolbarProps {
  layoutKey: string;
}

const TOOLBAR_BUTTONS_STYLE = { width: 66 };

const cellPositionInRows = (
  rows: table.Row[],
  cellKey: string,
): { x: number; y: number } | null => {
  for (let y = 0; y < rows.length; y++) {
    const x = rows[y].cells.indexOf(cellKey);
    if (x !== -1) return { x, y };
  }
  return null;
};

const Internal = ({ layoutKey }: ToolbarProps): ReactElement => {
  Base.useEnsureRetrieved({ key: layoutKey });
  const { name } = Layout.useSelectRequired(layoutKey);
  const editable = useSelectEditable(layoutKey);
  const selectedCellKeys = useSelectSelectedCellKeys(layoutKey);
  const rows = Base.useSelectRows({ key: layoutKey });
  const singleSelectedKey = selectedCellKeys.length === 1 ? selectedCellKeys[0] : null;
  const selectedCellPos = useMemo(
    () =>
      singleSelectedKey != null ? cellPositionInRows(rows, singleSelectedKey) : null,
    [singleSelectedKey, rows],
  );
  const handleExport = useExport();
  return (
    <Tb.Content>
      <Tb.Header>
        <Flex.Box x align="center">
          <Breadcrumb.Breadcrumb>
            <Breadcrumb.Segment weight={500} color={9} level="h5">
              <Icon.Table />
              {name}
            </Breadcrumb.Segment>
            {selectedCellPos != null && (
              <Breadcrumb.Segment color={8}>
                {Base.getCellColumn(selectedCellPos.x)}
                {selectedCellPos.y + 1}
              </Breadcrumb.Segment>
            )}
            {selectedCellKeys.length > 1 && (
              <Breadcrumb.Segment color={8}>
                {selectedCellKeys.length} cells
              </Breadcrumb.Segment>
            )}
          </Breadcrumb.Breadcrumb>
        </Flex.Box>
        <Flex.Box x style={TOOLBAR_BUTTONS_STYLE} empty>
          <Export.ToolbarButton onExport={() => handleExport(layoutKey)} />
          <Cluster.CopyLinkToolbarButton
            name={name}
            ontologyID={table.ontologyID(layoutKey)}
          />
        </Flex.Box>
      </Tb.Header>
      <Flex.Box full>
        {!editable ? (
          <NotEditableContent layoutKey={layoutKey} name={name} />
        ) : selectedCellKeys.length === 0 ? (
          <EmptyContent />
        ) : singleSelectedKey != null ? (
          <CellForm
            key={singleSelectedKey}
            tableKey={layoutKey}
            cellKey={singleSelectedKey}
          />
        ) : (
          <MultiCellForm tableKey={layoutKey} cellKeys={selectedCellKeys} />
        )}
      </Flex.Box>
    </Tb.Content>
  );
};

export const Toolbar = (props: ToolbarProps): ReactElement | null => {
  const pendingUpload = useSelectPendingUpload(props.layoutKey);
  return pendingUpload == null ? <Internal {...props} /> : null;
};

interface CellFormProps {
  tableKey: string;
  cellKey: string;
}

const CellForm = ({ tableKey, cellKey }: CellFormProps): ReactElement | null => {
  const cell = Base.useSelectCell({ key: tableKey, cellKey });
  const { dispatch } = Base.useDispatch();
  const store = useStore<RootState>();

  const handleVariantChange = useCallback(
    (variant: TableCells.Variant) => {
      if (cell == null || variant === cell.variant) return;
      const theme = Layout.selectTheme(store.getState());
      if (theme == null) throw new Error("Theme is null");
      const spec = TableCells.CELLS[variant];
      const nextProps = deep.overrideValidItems(
        cell.props,
        spec.defaultProps(theme),
        spec.schema,
      );
      dispatch({
        key: tableKey,
        actions: [table.setCell({ cell: { key: cellKey, variant, props: nextProps } })],
      });
    },
    [cell, cellKey, dispatch, store, tableKey],
  );

  const handleChange = useCallback(
    ({ values }: Form.OnChangeArgs<ReturnType<typeof record.unknownZ>>) => {
      if (cell == null) return;
      dispatch({
        key: tableKey,
        actions: [
          table.setCell({
            cell: { key: cellKey, variant: cell.variant, props: deep.copy(values) },
          }),
        ],
      });
    },
    [cell, cellKey, dispatch, tableKey],
  );

  const methods = Form.use<ReturnType<typeof record.unknownZ>>({
    values: cell != null ? deep.copy(cell.props) : {},
    schema: record.unknownZ(),
    onChange: handleChange,
    sync: true,
  });

  if (cell == null) return null;
  const F = TableCells.CELLS[cell.variant as keyof typeof TableCells.CELLS]?.Form;
  if (F == null) return null;
  return (
    <Form.Form<ReturnType<typeof record.unknownZ>> {...methods}>
      <F onVariantChange={handleVariantChange} />
    </Form.Form>
  );
};

const EmptyContent = (): ReactElement => (
  <Text.Text status="disabled" center>
    No cell selected. Select a cell to view its properties.
  </Text.Text>
);

interface NotEditableContentProps {
  layoutKey: string;
  name: string;
}

const NotEditableContent = ({
  layoutKey,
  name,
}: NotEditableContentProps): ReactElement => {
  const dispatch = useDispatch();
  const hasUpdatePermission = Access.useUpdateGranted(table.ontologyID(layoutKey));
  return (
    <EmptyAction
      x
      message={`${name} is not editable.${hasUpdatePermission ? " To make changes," : ""}`}
      action={hasUpdatePermission ? "enable editing." : undefined}
      onClick={() => dispatch(setEditable({ key: layoutKey, editable: true }))}
    />
  );
};

// MULTI_CELL_LEVEL_DEFAULT is the level shown in the multi-cell form when
// the selected cells disagree on level. Matches the value variant's default.
const MULTI_CELL_LEVEL_DEFAULT: text.Level = "p";

// colorPropOf returns the key in cell.props that stores the cell's
// fill/background color for its variant, or null if the variant has no such
// prop. Used by the multi-cell color groups so a single Color.Swatch can
// bulk-set the right prop across mixed variants.
const colorPropOf = (variant: string): "backgroundColor" | "color" | null => {
  if (variant === "text") return "backgroundColor";
  if (variant === "value") return "color";
  return null;
};

interface MultiCellFormProps {
  tableKey: string;
  cellKeys: string[];
}

const MultiCellForm = ({ tableKey, cellKeys }: MultiCellFormProps): ReactElement => {
  const cellsByKey = Base.useSelectCells({ key: tableKey, cellKeys });
  const { dispatch } = Base.useDispatch();
  const store = useStore<RootState>();

  // Bulk-apply a partial props patch across the selected cells. Cells with
  // no current entry in the store are skipped (the selection may include a
  // key from a removed row). The whole batch ships as one dispatch so undo
  // collapses to a single step.
  const applyPropPatch = useCallback(
    (
      patch: (cell: table.Cell) => Partial<record.Unknown> | null,
      onlyKeys?: string[],
    ) => {
      const targets = onlyKeys ?? cellKeys;
      const actions: table.Action[] = [];
      for (const key of targets) {
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
      if (actions.length === 0) return;
      dispatch({ key: tableKey, actions });
    },
    [cellKeys, cellsByKey, dispatch, tableKey],
  );

  // Distinct variants in the selection. When mixed, only the variant
  // selector is shown - format props don't align across variants.
  const variants = useMemo(() => {
    const s = new Set<string>();
    cellsByKey.forEach((c) => s.add(c.variant));
    return s;
  }, [cellsByKey]);
  const commonVariant =
    variants.size === 1
      ? ((variants.values().next().value ?? null) as TableCells.Variant | null)
      : null;

  const handleVariantChange = useCallback(
    (variant: TableCells.Variant) => {
      const theme = Layout.selectTheme(store.getState());
      if (theme == null) throw new Error("Theme is null");
      const spec = TableCells.CELLS[variant];
      const actions: table.Action[] = [];
      cellsByKey.forEach((cell, key) => {
        if (cell.variant === variant) return;
        const nextProps = deep.overrideValidItems(
          cell.props,
          spec.defaultProps(theme),
          spec.schema,
        );
        actions.push(table.setCell({ cell: { key, variant, props: nextProps } }));
      });
      if (actions.length === 0) return;
      dispatch({ key: tableKey, actions });
    },
    [cellsByKey, dispatch, store, tableKey],
  );

  // Group cells by their current fill/background color so each Color.Swatch
  // displays one of the colors actually in use; clicking a swatch updates
  // every cell that shared that color. Mirrors the schematic color-group
  // pattern.
  const colorGroups = useMemo(() => {
    const groups = new Map<color.Hex, string[]>();
    cellsByKey.forEach((cell, key) => {
      const propKey = colorPropOf(cell.variant);
      if (propKey == null) return;
      const raw = cell.props[propKey];
      if (raw == null) return;
      const hex = color.hex(raw as color.Crude);
      const existing = groups.get(hex);
      if (existing != null) existing.push(key);
      else groups.set(hex, [key]);
    });
    return groups;
  }, [cellsByKey]);

  const handleColorChange = useCallback(
    (groupKeys: string[], next: color.Color) =>
      applyPropPatch((cell) => {
        const propKey = colorPropOf(cell.variant);
        if (propKey == null) return null;
        return { [propKey]: next };
      }, groupKeys),
    [applyPropPatch],
  );

  // Level is shared across both variants. The displayed value is the first
  // cell's level so the dropdown shows something concrete even when cells
  // disagree; changing it bulk-applies to all selected cells.
  const firstLevel = useMemo(() => {
    for (const cell of cellsByKey.values()) {
      const lvl = cell.props.level;
      if (typeof lvl === "string") return lvl as text.Level;
    }
    return MULTI_CELL_LEVEL_DEFAULT;
  }, [cellsByKey]);

  const handleLevelChange = useCallback(
    (level: text.Level) => applyPropPatch(() => ({ level })),
    [applyPropPatch],
  );

  return (
    <Flex.Box x align="start" gap="large" style={MULTI_CELL_FORM_STYLE}>
      <Input.Item label="Variant" padHelpText={false}>
        <TableCells.SelectVariant
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
        <Select.Text.Level value={firstLevel} onChange={handleLevelChange} />
      </Input.Item>
    </Flex.Box>
  );
};

const MULTI_CELL_FORM_STYLE = { padding: "2rem" };
