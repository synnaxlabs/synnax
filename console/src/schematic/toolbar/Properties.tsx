// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import {
  Button,
  Color,
  Diagram,
  Direction,
  Divider,
  Flex,
  Form,
  Icon,
  Input,
  Schematic,
  Select,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import {
  box,
  color,
  deep,
  type dimensions,
  type direction,
  location,
  type record,
  xy,
} from "@synnaxlabs/x";
import { memo, type ReactElement, type ReactNode, useMemo } from "react";
import { useStore } from "react-redux";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { selectViewport, useSelectSelected } from "@/schematic/selectors";
import { createEditLayout } from "@/schematic/symbols/edit/Edit";
import { type RootState } from "@/store";

export interface PropertiesProps {
  layoutKey: string;
}

export const Properties = memo(({ layoutKey }: PropertiesProps): ReactElement => {
  const selected = useSelectSelected(layoutKey);
  const selectedConfigs = Schematic.useSelectConfigs({
    key: layoutKey,
    keys: selected,
  });
  if (selected.length === 0 || selectedConfigs.length == 0)
    return (
      <Text.Text status="disabled" center>
        Select a Schematic element to configure its properties.
      </Text.Text>
    );
  if (selected.length > 1)
    return <MultiConfig layoutKey={layoutKey} selectedConfigs={selectedConfigs} />;
  const elKey = selected[0];
  return <IndividualConfig key={elKey} layoutKey={layoutKey} elKey={elKey} />;
});
Properties.displayName = "PropertiesControls";

interface IndividualConfigProps {
  layoutKey: string;
  elKey: string;
}

const IndividualConfig = ({
  layoutKey,
  elKey,
}: IndividualConfigProps): ReactElement | null => {
  const config = Schematic.useSelectElementConfig({ key: layoutKey, elKey });
  const { dispatch } = Schematic.useDispatch();

  const onChange = (key: string, next: Schematic.ElementConfig): void => {
    dispatch({
      key: layoutKey,
      actions: schematic.setConfig({ key, config: next }),
    });
  };

  const formMethods = Form.use<typeof Schematic.elementConfigZ>({
    schema: Schematic.elementConfigZ,
    values: deep.copy(config),
    sync: true,
    onChange: ({ values }) => onChange(elKey, deep.copy(values)),
  });
  const specKey = Form.useFieldValue<string, string, typeof Schematic.elementConfigZ>(
    "specKey",
    { ctx: formMethods, optional: true },
  );
  const isRemote = schematic.symbol.keyZ.safeParse(specKey).success;
  const placeLayout = Layout.usePlacer();
  let actions: ReactNode = null;
  if (isRemote && specKey != null)
    actions = (
      <Button.Button
        variant="filled"
        size="tiny"
        className={CSS.BE("schematic", "properties", "edit")}
        onClick={() => placeLayout(createEditLayout({ args: { key: specKey } }))}
      >
        <Icon.Edit />
      </Button.Button>
    );

  if (config == null) return null;
  const C = Schematic.ELEMENT_REGISTRY[config.variant];

  return (
    <Flex.Box className={CSS.BE("schematic", "properties")} y>
      <Form.Form<typeof Schematic.elementConfigZ> {...formMethods}>
        <C.Form key={elKey} actions={actions} schematicKey={layoutKey} />
      </Form.Form>
    </Flex.Box>
  );
};

interface MultiElementPropertiesProps {
  layoutKey: string;
  selectedConfigs: Schematic.ElementConfig[];
}

const Multiconfig = ({
  layoutKey,
  selectedConfigs,
}: MultiElementPropertiesProps): ReactElement => {
  const handleError = Status.useErrorHandler();
  const selected = useSelectSelected(layoutKey);

  const selectedNodes = Schematic.useSelectNodes({ key: layoutKey, keys: selected });
  const { dispatch } = Schematic.useDispatch();
  const store = useStore<RootState>();

  const configByKey = useMemo(() => {
    const m = new Map<string, Schematic.ElementConfig>();
    selected.forEach((k, i) => {
      const cfg = selectedConfigs[i];
      if (cfg != null) m.set(k, cfg);
    });
    return m;
  }, [selected, selectedConfigs]);

  const nodesByKey = useMemo(() => {
    const m = new Map<string, schematic.Node>();
    selectedNodes.forEach((n) => m.set(n.key, n));
    return m;
  }, [selectedNodes]);

  const onChange = (elKey: string, next: Partial<Schematic.ElementConfig>): void => {
    const existing = (configByKey.get(elKey) ?? {}) as record.Unknown;
    dispatch({
      key: layoutKey,
      actions: schematic.setConfig({
        key: elKey,
        config: { ...existing, ...next } as record.Unknown,
      }),
    });
  };

  let firstNodeLabel: Schematic.Node.Label.Config | undefined;
  selectedConfigs.find((cfg) => {
    if (!("label" in cfg)) return false;
    firstNodeLabel = cfg.label as Schematic.Node.Label.Config | undefined;
    return firstNodeLabel != null;
  });

  const colorGroups = useMemo(() => {
    const groups: Record<color.Hex, string[]> = {};
    selected.forEach((key, i) => {
      const cfg = selectedConfigs[i];
      if (cfg == null || cfg.color == null) return;
      const hex = color.hex(cfg.color as color.Crude);
      if (!(hex in groups)) groups[hex] = [];
      groups[hex].push(key);
    });
    return groups;
  }, [selected, selectedConfigs]);

  const handleLayouts = (
    nodeEl: Element,
    nodeElBox: box.Box,
    zoom: number,
  ): Diagram.HandleLayout[] => {
    const handleEls = nodeEl.getElementsByClassName("react-flow__handle");
    return Array.from(handleEls).map((el) => {
      const pos = box.center(box.construct(el));
      const dist = xy.scale(xy.translation(box.topLeft(nodeElBox), pos), 1 / zoom);
      const match = el.className.match(/react-flow__handle-(\w+)/);
      if (match == null)
        throw new Error(`[schematic] - cannot find handle orientation`);
      const orientation = location.outerZ.parse(match[1]);
      return new Diagram.HandleLayout(dist, orientation);
    });
  };

  const getLayoutsForAlignment = () => {
    const zoom = selectViewport(store.getState(), layoutKey)?.zoom ?? 1;
    return selected
      .map((key) => {
        const node = nodesByKey.get(key);
        if (node == null) return null;
        try {
          const nodeEl = Diagram.selectNode(key);
          const nodeElBox = box.construct(nodeEl);
          const rect = nodeEl.getBoundingClientRect();
          const actualDims: dimensions.Dimensions = {
            width: rect.width / zoom,
            height: rect.height / zoom,
          };
          const nodeBox = box.construct(node.position, actualDims);
          return new Diagram.NodeLayout(
            key,
            nodeBox,
            handleLayouts(nodeEl, nodeElBox, zoom),
          );
        } catch (e) {
          handleError(e, "failed to calculate schematic node layout");
        }
        return null;
      })
      .filter((el) => el !== null);
  };

  const getLayoutsForDistribution = (): {
    layouts: Diagram.NodeLayout[];
    adjustPosition: (key: string, pos: xy.XY) => xy.XY;
  } => {
    const zoom = selectViewport(store.getState(), layoutKey)?.zoom ?? 1;
    const topOffsets = new Map<string, number>();
    const layouts = selected
      .map((key) => {
        const node = nodesByKey.get(key);
        if (node == null) return null;
        try {
          const nodeEl = Diagram.selectNode(key);
          const nodeElBox = box.construct(nodeEl);
          const rect = nodeEl.getBoundingClientRect();

          const gridItems = nodeEl.querySelectorAll(".pluto-grid__item");
          let minTop = rect.top;
          let maxBottom = rect.bottom;
          gridItems.forEach((item) => {
            const itemRect = item.getBoundingClientRect();
            minTop = Math.min(minTop, itemRect.top);
            maxBottom = Math.max(maxBottom, itemRect.bottom);
          });

          const actualDims = {
            width: rect.width / zoom,
            height: (maxBottom - minTop) / zoom,
          };

          const topExtension = (rect.top - minTop) / zoom;
          topOffsets.set(key, topExtension);
          const adjustedPosition = xy.translate(node.position, {
            x: 0,
            y: -topExtension,
          });

          const nodeBox = box.construct(adjustedPosition, actualDims);
          return new Diagram.NodeLayout(
            key,
            nodeBox,
            handleLayouts(nodeEl, nodeElBox, zoom),
          );
        } catch (e) {
          handleError(e, "failed to calculate schematic node layout");
        }
        return null;
      })
      .filter((el) => el !== null);

    const adjustPosition = (key: string, pos: xy.XY): xy.XY => {
      const topOffset = topOffsets.get(key) ?? 0;
      return xy.translate(pos, { x: 0, y: topOffset });
    };
    return { layouts, adjustPosition };
  };

  const applyNodePositions = (layouts: Diagram.NodeLayout[]): void => {
    if (layouts.length === 0) return;
    dispatch({
      key: layoutKey,
      actions: layouts.map((n) =>
        schematic.setNodePosition({ key: n.key, position: box.topLeft(n.box) }),
      ),
    });
  };

  const handleAlignToLocation = (loc: location.Outer): void => {
    applyNodePositions(Diagram.alignNodesToLocation(getLayoutsForAlignment(), loc));
  };

  const handleAlignAlongDirection = (dir: direction.Direction): void => {
    applyNodePositions(Diagram.alignNodesAlongDirection(getLayoutsForAlignment(), dir));
  };

  const handleDistribute = (dir: direction.Direction): void => {
    const { layouts, adjustPosition } = getLayoutsForDistribution();
    const distributed = Diagram.distributeNodes(layouts, dir);
    const adjusted = distributed.map((n) => {
      const pos = adjustPosition(n.key, box.topLeft(n.box));
      return new Diagram.NodeLayout(
        n.key,
        box.construct(pos, box.dims(n.box)),
        n.handles,
      );
    });
    applyNodePositions(adjusted);
  };

  const handleRotateIndividual = (dir: direction.Angular): void => {
    selectedConfigs.forEach((cfg, i) => {
      if (!("orientation" in cfg) || cfg.orientation == null) return;
      onChange(selected[i], {
        orientation: location.rotate(cfg.orientation, dir),
      } as Partial<Schematic.ElementConfig>);
    });
  };

  const handleRotateGroup = (dir: direction.Angular): void => {
    applyNodePositions(Diagram.rotateNodesAroundCenter(getLayoutsForAlignment(), dir));
    handleRotateIndividual(dir);
  };

  const handleLabelProp = <K extends keyof Schematic.Node.Label.Config>(
    key: K,
    value: Schematic.Node.Label.Config[K],
  ): void => {
    selectedConfigs.forEach((cfg, i) => {
      if (!("label" in cfg) || cfg.label == null) return;
      onChange(selected[i], {
        label: { ...(cfg.label as Schematic.Node.Label.Config), [key]: value },
      } as Partial<Schematic.ElementConfig>);
    });
  };

  return (
    <Flex.Box
      align="start"
      x
      className={CSS.BE("schematic", "properties", "multi")}
      gap="large"
    >
      <Input.Item label="Selection Colors" align="start">
        <Flex.Box x>
          {Object.entries(colorGroups).map(([hex, keys]) => (
            <Color.Swatch
              key={keys[0]}
              value={hex}
              onChange={(c: color.Color) => {
                keys.forEach((key) =>
                  onChange(key, { color: c } as Partial<Schematic.ElementConfig>),
                );
              }}
            />
          ))}
        </Flex.Box>
      </Input.Item>
      <Input.Item label="Align">
        <Flex.Box x>
          <Button.Button
            tooltip="Align symbols vertically"
            onClick={() => handleAlignAlongDirection("x")}
          >
            <Icon.Align.YCenter />
          </Button.Button>
          <Button.Button
            tooltip="Align symbols horizontally"
            onClick={() => handleAlignAlongDirection("y")}
          >
            <Icon.Align.XCenter />
          </Button.Button>
          <Divider.Divider direction="y" />
          <Button.Button
            tooltip="Align symbols left"
            onClick={() => handleAlignToLocation("left")}
          >
            <Icon.Align.Left />
          </Button.Button>
          <Button.Button
            tooltip="Align symbols top"
            onClick={() => handleAlignToLocation("top")}
          >
            <Icon.Align.Top />
          </Button.Button>
          <Button.Button
            tooltip="Align symbols bottom"
            onClick={() => handleAlignToLocation("bottom")}
          >
            <Icon.Align.Bottom />
          </Button.Button>
          <Button.Button
            tooltip="Align symbols right"
            onClick={() => handleAlignToLocation("right")}
          >
            <Icon.Align.Right />
          </Button.Button>
        </Flex.Box>
      </Input.Item>
      {selected.length >= 3 && (
        <Input.Item label="Spacing">
          <Flex.Box x>
            <Button.Button
              tooltip="Distribute symbol spacing horizontally"
              onClick={() => handleDistribute("x")}
            >
              <Icon.Distribute.X />
            </Button.Button>
            <Button.Button
              tooltip="Distribute symbol spacing vertically"
              onClick={() => handleDistribute("y")}
            >
              <Icon.Distribute.Y />
            </Button.Button>
          </Flex.Box>
        </Input.Item>
      )}
      <Input.Item label="Rotate">
        <Flex.Box x>
          <Button.Button
            tooltip="Rotate symbols clockwise"
            onClick={() => handleRotateIndividual("clockwise")}
          >
            <Icon.RotateGroup.CW />
          </Button.Button>
          <Button.Button
            tooltip="Rotate symbols counterclockwise"
            onClick={() => handleRotateIndividual("counterclockwise")}
          >
            <Icon.RotateGroup.CCW />
          </Button.Button>
        </Flex.Box>
      </Input.Item>
      <Input.Item label="Rotate Selection">
        <Flex.Box x>
          <Button.Button
            tooltip="Rotate selection clockwise"
            onClick={() => handleRotateGroup("clockwise")}
          >
            <Icon.RotateAroundCenter.CW />
          </Button.Button>
          <Button.Button
            tooltip="Rotate selection counterclockwise"
            onClick={() => handleRotateGroup("counterclockwise")}
          >
            <Icon.RotateAroundCenter.CCW />
          </Button.Button>
        </Flex.Box>
      </Input.Item>
      <Input.Item label="Label Wrap Width" align="start">
        <Input.Numeric
          value={firstNodeLabel?.maxInlineSize ?? 150}
          onChange={(v) => handleLabelProp("maxInlineSize", v)}
          endContent="px"
        />
      </Input.Item>
      <Input.Item label="Label Size" align="start">
        <Select.Text.Level
          value={firstNodeLabel?.level ?? "p"}
          onChange={(v: Text.Level) => handleLabelProp("level", v)}
        />
      </Input.Item>
      <Input.Item label="Label Alignment" align="start">
        <Select.Flex.Alignment
          value={firstNodeLabel?.align ?? "center"}
          onChange={(v: Flex.Alignment) => handleLabelProp("align", v)}
        />
      </Input.Item>
      <Input.Item label="Label Direction" align="start">
        <Direction.Select
          value={firstNodeLabel?.direction ?? "x"}
          onChange={(v: direction.Direction) => handleLabelProp("direction", v)}
          yDirection="down"
        />
      </Input.Item>
      <Input.Item label="Label Orientation" align="start">
        <Schematic.Node.Orientation.Select
          value={{ inner: "top", outer: firstNodeLabel?.orientation ?? "top" }}
          onChange={(v) =>
            v.outer !== "center" && handleLabelProp("orientation", v.outer)
          }
          hideInner
        />
      </Input.Item>
    </Flex.Box>
  );
};
