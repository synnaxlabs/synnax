// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/service/table/Table.css";

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
  Text,
  Theming,
} from "@synnaxlabs/pluto";
import { color, deep, record, type text } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { EmptyAction, Toolbar as Tb } from "@/component";
import { CSS } from "@/component/css";
import { Export } from "@/export";
import { useExport } from "@/service/table/export";
import { Session } from "@/session";

export interface ToolbarProps {
  layoutKey: string;
}

const Internal = (): ReactElement => {
  const key = Base.useKey();
  const name = Base.useSelectName();
  const editable = Session.Table.useSelectEditable();
  const selectedCellKeys = Session.Table.useSelectSelectedCellKeys();
  const cellsByKey = Base.useSelectCells({ cellKeys: selectedCellKeys });
  const liveCellCount = cellsByKey.size;
  const singleSelectedKey =
    liveCellCount === 1 ? (cellsByKey.keys().next().value ?? null) : null;
  const selectedCellPos = Base.useCellPosition({ cellKey: singleSelectedKey ?? "" });
  const handleExport = useExport();
  return (
    <Tb.Content>
      <Tb.Header>
        <Flex.Box x align="center">
          <Breadcrumb.Breadcrumb>
            <Breadcrumb.Segment weight={500} color={10} level="h5">
              <Icon.Table />
              {name}
            </Breadcrumb.Segment>
            {selectedCellPos != null && (
              <Breadcrumb.Segment color={8}>
                {Base.getCellColumn(selectedCellPos.x)}
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
      </Tb.Header>
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
    </Tb.Content>
  );
};

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement => (
  <Base.Suspended tableKey={layoutKey}>
    <Internal />
  </Base.Suspended>
);

// buildVariantSwapActions returns one setCell action per cell whose variant
// differs from the target. Compatible fields survive the swap.
const buildVariantSwapActions = (
  cells: Iterable<[string, Base.Cell.Config]>,
  variant: Base.Cell.Variant,
  theme: Theming.Theme,
): table.Action[] => {
  const spec = Base.Cell.REGISTRY[variant];
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
  const cell = Base.useSelectCell({ cellKey });
  const dispatch = Base.useSingleDispatch();
  const theme = Theming.use();

  const handleVariantChange = useCallback(
    (variant: Base.Cell.Variant) => {
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
  const C = Base.Cell.REGISTRY[cell.variant];
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
  const key = Base.useKey();
  const dispatch = useDispatch();
  const hasUpdatePermission = Access.useUpdateGranted(table.ontologyID(key));
  return (
    <EmptyAction
      x
      message={`${name} is not editable.${hasUpdatePermission ? " To make changes," : ""}`}
      action={hasUpdatePermission ? "enable editing." : undefined}
      onClick={() => dispatch(Session.Table.setEditable({ key, editable: true }))}
    />
  );
};

const readCellColor = (cell: Base.Cell.Config): color.Hex | null => {
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
  cell: Base.Cell.Config,
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
  const cellsByKey = Base.useSelectCells({ cellKeys });
  const dispatch = Base.useSingleDispatch();
  const theme = Theming.use();

  // Cells absent from the store are skipped (selection may include keys from
  // a removed row). One dispatch per call so undo collapses to one step.
  const applyPropPatch = useCallback(
    (
      keys: string[],
      patch: (cell: Base.Cell.Config) => Partial<record.Unknown> | null,
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
    const s = new Set<Base.Cell.Variant>();
    cellsByKey.forEach((c) => s.add(c.variant));
    return s;
  }, [cellsByKey]);
  const commonVariant =
    variants.size === 1 ? (variants.values().next().value ?? null) : null;

  const handleVariantChange = useCallback(
    (variant: Base.Cell.Variant) => {
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
        <Base.Cell.SelectVariant
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
