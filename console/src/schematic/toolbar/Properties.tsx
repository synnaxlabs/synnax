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
import { box, color, deep, type direction, location, xy } from "@synnaxlabs/x";
import { memo, type ReactElement, type ReactNode } from "react";
import { useDispatch, useStore } from "react-redux";

import { Layout } from "@/layout";
import { expandGroupPositions, resolveAlignmentKey } from "@/schematic/groups";
import {
  type ElementInfo,
  selectRequired,
  selectViewport,
  useSelectRequiredEdge,
  useSelectRequiredNodeProps,
  useSelectSelectedElementDigests,
  useSelectSelectedElementsProps,
} from "@/schematic/selectors";
import {
  groupSelection,
  setElementProps,
  setNodePositions,
  ungroupSelection,
} from "@/schematic/slice";
import { createEditLayout } from "@/schematic/symbols/edit/Edit";
import { type EdgeProps, type NodeProps } from "@/schematic/types";
import { type nodePropsZ } from "@/schematic/types/v0";
import { type RootState } from "@/store";

export interface PropertiesProps {
  layoutKey: string;
}

export const PropertiesControls = memo(
  ({ layoutKey }: PropertiesProps): ReactElement => {
    const digests = useSelectSelectedElementDigests(layoutKey);
    if (digests.length === 0)
      return (
        <Text.Text status="disabled" center>
          Select a Schematic element to configure its properties.
        </Text.Text>
      );

    if (digests.length > 1) return <MultiElementProperties layoutKey={layoutKey} />;

    const selected = digests[0];

    if (selected.type === "edge")
      return <EdgeProperties layoutKey={layoutKey} edgeKey={selected.key} />;
    return (
      <IndividualProperties
        key={selected.key}
        layoutKey={layoutKey}
        nodeKey={selected.key}
      />
    );
  },
);
PropertiesControls.displayName = "PropertiesControls";

interface IndividualPropertiesProps {
  layoutKey: string;
  nodeKey: string;
}

const IndividualProperties = ({
  layoutKey,
  nodeKey,
}: IndividualPropertiesProps): ReactElement | null => {
  const props = useSelectRequiredNodeProps(layoutKey, nodeKey);
  const C = Schematic.Symbol.REGISTRY[props.key];
  const dispatch = useDispatch();

  const onChange = (key: string, props: NodeProps): void => {
    dispatch(setElementProps({ layoutKey, key, props }));
  };

  const formMethods = Form.use<typeof nodePropsZ>({
    values: deep.copy(props),
    sync: true,
    onChange: ({ values }) => onChange(nodeKey, deep.copy(values)),
  });
  const specKey = Form.useFieldValue<string, string, typeof nodePropsZ>("specKey", {
    ctx: formMethods,
    optional: true,
  });
  const isRemote = schematic.symbol.keyZ.safeParse(specKey).success;
  const actions: ReactNode[] = [];
  const placeLayout = Layout.usePlacer();
  if (isRemote && specKey != null)
    actions.push(
      <Button.Button
        key="edit"
        variant="filled"
        size="tiny"
        style={{ marginRight: "1rem" }}
        onClick={() => placeLayout(createEditLayout({ args: { key: specKey } }))}
      >
        <Icon.Edit />
      </Button.Button>,
    );
  if (props.key === "group")
    actions.push(
      <Input.Item key="ungroup" label="Grouping">
        <Button.Button onClick={() => dispatch(ungroupSelection({ key: layoutKey }))}>
          <Icon.Ungroup />
          Ungroup
        </Button.Button>
      </Input.Item>,
    );

  return (
    <Flex.Box style={{ height: "100%" }} y>
      <Form.Form<typeof nodePropsZ> {...formMethods}>
        <C.Form
          {...formMethods}
          key={nodeKey}
          actions={actions}
          schematicKey={layoutKey}
        />
      </Form.Form>
    </Flex.Box>
  );
};

interface EdgePropertiesProps {
  layoutKey: string;
  edgeKey: string;
}

const SELECT_EDGE_TYPE_STYLE: React.CSSProperties = {
  width: "25rem",
};

const LABEL_WRAP_WIDTH_STYLE: React.CSSProperties = {
  width: 125,
};

const EdgeProperties = ({
  layoutKey,
  edgeKey,
}: EdgePropertiesProps): ReactElement | null => {
  const edge = useSelectRequiredEdge(layoutKey, edgeKey);
  const dispatch = useDispatch();
  const onChange = (key: string, props: Partial<EdgeProps>): void => {
    dispatch(setElementProps({ layoutKey, key, props }));
  };
  return (
    <Flex.Box style={{ padding: "2rem" }} align="start" x>
      <Input.Item label="Color" align="start">
        <Color.Swatch
          value={(edge.data?.color ?? color.ZERO) as color.Crude}
          onChange={(v: color.Color) => {
            onChange(edge.key, { color: color.hex(v) });
          }}
        />
      </Input.Item>
      <Input.Item label="Type" align="start">
        <Schematic.SelectEdgeType
          value={edge.data?.variant as Schematic.EdgeType}
          onChange={(variant: Schematic.EdgeType) => onChange(edge.key, { variant })}
          style={SELECT_EDGE_TYPE_STYLE}
        />
      </Input.Item>
    </Flex.Box>
  );
};

interface MultiElementPropertiesProps {
  layoutKey: string;
}

const MultiElementProperties = ({
  layoutKey,
}: MultiElementPropertiesProps): ReactElement => {
  const handleError = Status.useErrorHandler();
  const elements = useSelectSelectedElementsProps(layoutKey);
  const dispatch = useDispatch();
  const onChange = (key: string, props: Partial<NodeProps>): void => {
    dispatch(setElementProps({ layoutKey, key, props }));
  };

  const colorGroups: Record<string, ElementInfo[]> = {};
  elements.forEach((e) => {
    let colorVal: color.Color | null = null;
    if (e.type === "edge") colorVal = color.colorZ.parse(e.edge.data?.color);
    else if (e.props.color != null) colorVal = color.construct(e.props.color);
    if (colorVal === null) return;
    const hex = color.hex(colorVal);
    if (!(hex in colorGroups)) colorGroups[hex] = [];
    colorGroups[hex].push(e);
  });

  const firstNode = elements.find((e) => e.type === "node");
  const firstNodeLabel = firstNode?.props.label;

  const store = useStore<RootState>();

  const extractHandles = (
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
      return new Diagram.HandleLayout(dist, location.outerZ.parse(match[1]));
    });
  };

  const measureNodeLayout = (
    key: string,
    position: xy.XY,
    zoom: number,
    opts?: { includeGridExtent?: boolean },
  ): { layout: Diagram.NodeLayout; topOffset: number } | null => {
    try {
      const nodeEl = Diagram.selectNode(key);
      const nodeElBox = box.construct(nodeEl);
      const rect = nodeEl.getBoundingClientRect();
      let minTop = rect.top;
      let maxBottom = rect.bottom;
      if (opts?.includeGridExtent)
        nodeEl.querySelectorAll(".pluto-grid__item").forEach((item) => {
          const itemRect = item.getBoundingClientRect();
          minTop = Math.min(minTop, itemRect.top);
          maxBottom = Math.max(maxBottom, itemRect.bottom);
        });
      const topExtension = (rect.top - minTop) / zoom;
      const actualDims = {
        width: rect.width / zoom,
        height: (maxBottom - minTop) / zoom,
      };
      const adjustedPosition = xy.translate(position, {
        x: 0,
        y: -topExtension,
      });
      const nodeBox = box.construct(adjustedPosition, actualDims);
      const handles = extractHandles(nodeEl, nodeElBox, zoom);
      return {
        layout: new Diagram.NodeLayout(key, nodeBox, handles),
        topOffset: topExtension,
      };
    } catch (e) {
      handleError(e, "failed to calculate schematic node layout");
      return null;
    }
  };

  const getSchematicContext = () => {
    const viewport = selectViewport(store.getState(), layoutKey);
    const { nodes: allNodes, props: allProps } = selectRequired(
      store.getState(),
      layoutKey,
    );
    return { zoom: viewport?.zoom ?? 1, allNodes, allProps };
  };

  const getLayoutsForAlignment = () => {
    const { zoom, allNodes, allProps } = getSchematicContext();
    const seenKeys = new Set<string>();
    return elements
      .map((el) => {
        if (el.type !== "node") return null;
        const resolved = resolveAlignmentKey(
          el.key,
          allProps,
          allNodes,
          el.node.position,
        );
        if (seenKeys.has(resolved.key)) return null;
        seenKeys.add(resolved.key);
        return measureNodeLayout(resolved.key, resolved.position, zoom)?.layout ?? null;
      })
      .filter((el) => el !== null);
  };

  const getLayoutsForDistribution = (): {
    layouts: Diagram.NodeLayout[];
    adjustPosition: (key: string, pos: xy.XY) => xy.XY;
  } => {
    const { zoom, allNodes, allProps } = getSchematicContext();
    const topOffsets = new Map<string, number>();
    const seenKeys = new Set<string>();
    const layouts = elements
      .map((el) => {
        if (el.type !== "node") return null;
        const resolved = resolveAlignmentKey(
          el.key,
          allProps,
          allNodes,
          el.node.position,
        );
        if (seenKeys.has(resolved.key)) return null;
        seenKeys.add(resolved.key);
        const result = measureNodeLayout(resolved.key, resolved.position, zoom, {
          includeGridExtent: true,
        });
        if (result == null) return null;
        topOffsets.set(resolved.key, result.topOffset);
        return result.layout;
      })
      .filter((el) => el !== null);

    const adjustPosition = (key: string, pos: xy.XY): xy.XY => {
      const topOffset = topOffsets.get(key) ?? 0;
      return xy.translate(pos, { x: 0, y: topOffset });
    };

    return { layouts, adjustPosition };
  };

  const applyNodePositions = (layouts: Diagram.NodeLayout[]): void => {
    const { nodes, props } = selectRequired(store.getState(), layoutKey);
    const raw: [string, xy.XY][] = layouts.map((n) => [n.key, box.topLeft(n.box)]);
    const positions = expandGroupPositions(raw, nodes, props);
    dispatch(setNodePositions({ key: layoutKey, positions }));
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
    elements.forEach((el) => {
      if (el.type !== "node") return;
      const parsed = location.outerZ.safeParse(el.props.orientation);
      if (!parsed.success) return;
      onChange(el.key, { orientation: location.rotate(parsed.data, dir) });
    });
  };

  const handleRotateGroup = (dir: direction.Angular): void => {
    applyNodePositions(Diagram.rotateNodesAroundCenter(getLayoutsForAlignment(), dir));
    handleRotateIndividual(dir);
  };

  const handleLabelProp = <K extends keyof Schematic.Symbol.LabelExtensionProps>(
    key: K,
    value: Schematic.Symbol.LabelExtensionProps[K],
  ): void => {
    elements.forEach((e) => {
      if (e.type !== "node" || e.props.label == null) return;
      onChange(e.key, { label: { ...e.props.label, [key]: value } });
    });
  };

  return (
    <Flex.Box align="start" x style={{ padding: "2rem" }} gap="large">
      <Flex.Box align="start" x wrap gap="large" grow>
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
        {elements.length >= 3 && (
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
        <Input.Item label="Pivot">
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
        <Input.Item label="Grouping">
          <Flex.Box x>
            <Button.Button
              tooltip="Group selected symbols (Ctrl+G)"
              onClick={() =>
                dispatch(
                  groupSelection({
                    key: layoutKey,
                    props: {
                      key: "group",
                      ...Schematic.Symbol.REGISTRY.group.defaultProps({} as never),
                    },
                  }),
                )
              }
            >
              <Icon.Group />
            </Button.Button>
            <Button.Button
              tooltip="Ungroup selected symbols (Ctrl+Shift+G)"
              onClick={() => dispatch(ungroupSelection({ key: layoutKey }))}
            >
              <Icon.Ungroup />
            </Button.Button>
          </Flex.Box>
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
        <Input.Item label="Label Wrap Width" align="start">
          <Input.Numeric
            value={firstNodeLabel?.maxInlineSize ?? 150}
            onChange={(v) => handleLabelProp("maxInlineSize", v)}
            endContent="px"
            style={LABEL_WRAP_WIDTH_STYLE}
          />
        </Input.Item>
        <Input.Item label="Selection Colors" align="start">
          <Flex.Box x>
            {Object.entries(colorGroups).map(([hex, elements]) => (
              <Color.Swatch
                key={elements[0].key}
                value={hex}
                onChange={(v: color.Color) => {
                  elements.forEach((e) => onChange(e.key, { color: color.hex(v) }));
                }}
              />
            ))}
          </Flex.Box>
        </Input.Item>
      </Flex.Box>
      <Input.Item label="Label Orientation" align="start">
        <Schematic.Symbol.SelectOrientation
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
