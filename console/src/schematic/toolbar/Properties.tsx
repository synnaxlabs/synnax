// Copyright 2025 Synnax Labs, Inc.
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
  Divider,
  Flex,
  Form,
  Icon,
  Input,
  Schematic,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { box, color, deep, location, xy } from "@synnaxlabs/x";
import { memo, type ReactElement, type ReactNode } from "react";
import { useDispatch, useStore } from "react-redux";

import { Layout } from "@/layout";
import {
  type ElementInfo,
  selectViewport,
  useSelectRequiredEdge,
  useSelectRequiredNodeProps,
  useSelectSelectedElementDigests,
  useSelectSelectedElementsProps,
} from "@/schematic/selectors";
import { setElementProps, setNodePositions } from "@/schematic/slice";
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
  let actions: ReactNode = null;
  const placeLayout = Layout.usePlacer();
  if (isRemote && specKey != null)
    actions = (
      <Button.Button
        variant="filled"
        size="tiny"
        style={{ marginRight: "1rem" }}
        onClick={() => placeLayout(createEditLayout({ args: { key: specKey } }))}
      >
        <Icon.Edit />
      </Button.Button>
    );

  return (
    <Flex.Box style={{ height: "100%" }} y>
      <Form.Form<typeof nodePropsZ> {...formMethods}>
        <C.Form {...formMethods} key={nodeKey} actions={actions} />
      </Form.Form>
    </Flex.Box>
  );
};

interface EdgePropertiesProps {
  layoutKey: string;
  edgeKey: string;
}

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

  const store = useStore<RootState>();

  const getLayoutsForAlignment = () => {
    const viewport = selectViewport(store.getState(), layoutKey);

    // For alignment: use uniform bottom padding so all nodes align at same visual level
    let maxBottomExtension = 0;
    elements.forEach((el) => {
      if (el.type !== "node") return;
      try {
        const nodeEl = Diagram.selectNode(el.key);
        const rect = nodeEl.getBoundingClientRect();
        const bottomIndicator = nodeEl.querySelector(
          ".pluto-grid__item.pluto--location-bottom",
        );
        if (bottomIndicator) {
          const indicatorRect = bottomIndicator.getBoundingClientRect();
          const extensionBelow = Math.max(0, indicatorRect.bottom - rect.bottom);
          maxBottomExtension = Math.max(
            maxBottomExtension,
            extensionBelow / (viewport?.zoom ?? 1),
          );
        }
      } catch (_e) {
        // Skip on error
      }
    });

    return elements
      .map((el) => {
        if (el.type !== "node") return null;
        try {
          const nodeEl = Diagram.selectNode(el.key);
          const nodeElBox = box.construct(nodeEl);
          const rect = nodeEl.getBoundingClientRect();

          const actualDims = {
            width: rect.width / (viewport?.zoom ?? 1),
            height: rect.height / (viewport?.zoom ?? 1),
          };

          // Add uniform padding for consistent alignment
          actualDims.height += maxBottomExtension;

          const nodeBox = box.construct(el.node.position, actualDims);
          const handleEls = nodeEl.getElementsByClassName("react-flow__handle");
          const handles = Array.from(handleEls).map((el) => {
            const pos = box.center(box.construct(el));
            const dist = xy.scale(
              xy.translation(box.topLeft(nodeElBox), pos),
              1 / (viewport?.zoom ?? 1),
            );
            const match = el.className.match(/react-flow__handle-(\w+)/);
            if (match == null)
              throw new Error(`[schematic] - cannot find handle orientation`);
            const orientation = location.construct(match[1]) as location.Outer;
            return new Diagram.HandleLayout(dist, orientation);
          });
          return new Diagram.NodeLayout(el.key, nodeBox, handles);
        } catch (e) {
          handleError(e, "failed to calculate schematic node layout");
        }
        return null;
      })
      .filter((el) => el !== null);
  };

  // Track top offsets for distribution (to reverse adjustment when setting positions)
  const topOffsetsForDistribution = new Map<string, number>();

  const getLayoutsForDistribution = () => {
    const viewport = selectViewport(store.getState(), layoutKey);
    topOffsetsForDistribution.clear();

    // For distribution: use actual extensions to calculate true visual extents
    return elements
      .map((el) => {
        if (el.type !== "node") return null;
        try {
          const nodeEl = Diagram.selectNode(el.key);
          const nodeElBox = box.construct(nodeEl);
          const rect = nodeEl.getBoundingClientRect();

          // Calculate union of all child elements (labels, indicators, etc.)
          const gridItems = nodeEl.querySelectorAll(".pluto-grid__item");
          let minTop = rect.top;
          let maxBottom = rect.bottom;

          gridItems.forEach((item) => {
            const itemRect = item.getBoundingClientRect();
            minTop = Math.min(minTop, itemRect.top);
            maxBottom = Math.max(maxBottom, itemRect.bottom);
          });

          const actualDims = {
            width: rect.width / (viewport?.zoom ?? 1),
            height: (maxBottom - minTop) / (viewport?.zoom ?? 1),
          };

          // Adjust position if there are top extensions
          const topExtension = (rect.top - minTop) / (viewport?.zoom ?? 1);
          topOffsetsForDistribution.set(el.key, topExtension);
          const adjustedPosition = xy.translate(el.node.position, {
            x: 0,
            y: -topExtension,
          });

          const nodeBox = box.construct(adjustedPosition, actualDims);
          const handleEls = nodeEl.getElementsByClassName("react-flow__handle");
          const handles = Array.from(handleEls).map((el) => {
            const pos = box.center(box.construct(el));
            const dist = xy.scale(
              xy.translation(box.topLeft(nodeElBox), pos),
              1 / (viewport?.zoom ?? 1),
            );
            const match = el.className.match(/react-flow__handle-(\w+)/);
            if (match == null)
              throw new Error(`[schematic] - cannot find handle orientation`);
            const orientation = location.construct(match[1]) as location.Outer;
            return new Diagram.HandleLayout(dist, orientation);
          });
          return new Diagram.NodeLayout(el.key, nodeBox, handles);
        } catch (e) {
          handleError(e, "failed to calculate schematic node layout");
        }
        return null;
      })
      .filter((el) => el !== null);
  };

  return (
    <Flex.Box align="start" x style={{ padding: "2rem" }} gap="large">
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
      <Input.Item label="Align">
        <Flex.Box x>
          <Button.Button
            tooltip="Align nodes vertically"
            onClick={() => {
              const newPositions = Diagram.alignNodes(getLayoutsForAlignment(), "x");
              dispatch(
                setNodePositions({
                  key: layoutKey,
                  positions: Object.fromEntries(
                    newPositions.map((n) => [n.key, box.topLeft(n.box)]),
                  ),
                }),
              );
            }}
          >
            <Icon.Align.YCenter />
          </Button.Button>
          <Button.Button
            tooltip="Align nodes horizontally"
            onClick={() => {
              const newPositions = Diagram.alignNodes(getLayoutsForAlignment(), "y");
              dispatch(
                setNodePositions({
                  key: layoutKey,
                  positions: Object.fromEntries(
                    newPositions.map((n) => [n.key, box.topLeft(n.box)]),
                  ),
                }),
              );
            }}
          >
            <Icon.Align.XCenter />
          </Button.Button>
          <Divider.Divider direction="y" />
          <Button.Button
            tooltip="Align nodes left"
            onClick={() => {
              const newPositions = Diagram.alignNodes(getLayoutsForAlignment(), "left");
              dispatch(
                setNodePositions({
                  key: layoutKey,
                  positions: Object.fromEntries(
                    newPositions.map((n) => [n.key, box.topLeft(n.box)]),
                  ),
                }),
              );
            }}
          >
            <Icon.Align.Left />
          </Button.Button>
          <Button.Button
            tooltip="Align nodes top"
            onClick={() => {
              const newPositions = Diagram.alignNodes(getLayoutsForAlignment(), "top");
              dispatch(
                setNodePositions({
                  key: layoutKey,
                  positions: Object.fromEntries(
                    newPositions.map((n) => [n.key, box.topLeft(n.box)]),
                  ),
                }),
              );
            }}
          >
            <Icon.Align.Top />
          </Button.Button>
          <Button.Button
            tooltip="Align nodes bottom"
            onClick={() => {
              const newPositions = Diagram.alignNodes(
                getLayoutsForAlignment(),
                "bottom",
              );
              dispatch(
                setNodePositions({
                  key: layoutKey,
                  positions: Object.fromEntries(
                    newPositions.map((n) => [n.key, box.topLeft(n.box)]),
                  ),
                }),
              );
            }}
          >
            <Icon.Align.Bottom />
          </Button.Button>
          <Button.Button
            tooltip="Align nodes right"
            onClick={() => {
              const newPositions = Diagram.alignNodes(
                getLayoutsForAlignment(),
                "right",
              );
              dispatch(
                setNodePositions({
                  key: layoutKey,
                  positions: Object.fromEntries(
                    newPositions.map((n) => [n.key, box.topLeft(n.box)]),
                  ),
                }),
              );
            }}
          >
            <Icon.Align.Right />
          </Button.Button>
        </Flex.Box>
      </Input.Item>
      <Input.Item label="Spacing">
        <Flex.Box x>
          <Button.Button
            tooltip="Distribute nodes horizontally"
            onClick={() => {
              const newPositions = Diagram.distributeNodes(
                getLayoutsForDistribution(),
                "horizontal",
              );
              dispatch(
                setNodePositions({
                  key: layoutKey,
                  positions: Object.fromEntries(
                    newPositions.map((n) => {
                      const topOffset = topOffsetsForDistribution.get(n.key) ?? 0;
                      const adjustedPos = box.topLeft(n.box);
                      return [n.key, xy.translate(adjustedPos, { x: 0, y: topOffset })];
                    }),
                  ),
                }),
              );
            }}
          >
            <Icon.Distribute.X />
          </Button.Button>
          <Button.Button
            tooltip="Distribute nodes vertically"
            onClick={() => {
              const newPositions = Diagram.distributeNodes(
                getLayoutsForDistribution(),
                "vertical",
              );
              dispatch(
                setNodePositions({
                  key: layoutKey,
                  positions: Object.fromEntries(
                    newPositions.map((n) => {
                      const topOffset = topOffsetsForDistribution.get(n.key) ?? 0;
                      const adjustedPos = box.topLeft(n.box);
                      return [n.key, xy.translate(adjustedPos, { x: 0, y: topOffset })];
                    }),
                  ),
                }),
              );
            }}
          >
            <Icon.Distribute.Y />
          </Button.Button>
        </Flex.Box>
      </Input.Item>
      <Input.Item label="Rotate">
        <Flex.Box x>
          <Button.Button
            tooltip="Rotate nodes clockwise"
            onClick={() => {
              const layouts = getLayoutsForAlignment();
              const newPositions = Diagram.rotateNodes(layouts, "clockwise");
              dispatch(
                setNodePositions({
                  key: layoutKey,
                  positions: Object.fromEntries(
                    newPositions.map((n) => [n.key, box.topLeft(n.box)]),
                  ),
                }),
              );
            }}
          >
            <Icon.RotateGroup.CW />
          </Button.Button>
          <Button.Button
            tooltip="Rotate nodes counter-clockwise"
            onClick={() => {
              const layouts = getLayoutsForAlignment();
              const newPositions = Diagram.rotateNodes(layouts, "counterclockwise");
              dispatch(
                setNodePositions({
                  key: layoutKey,
                  positions: Object.fromEntries(
                    newPositions.map((n) => [n.key, box.topLeft(n.box)]),
                  ),
                }),
              );
            }}
          >
            <Icon.RotateGroup.CCW />
          </Button.Button>
        </Flex.Box>
      </Input.Item>
    </Flex.Box>
  );
};
