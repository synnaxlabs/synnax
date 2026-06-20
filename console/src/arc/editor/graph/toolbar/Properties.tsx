// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import {
  Arc,
  Button,
  Diagram,
  Flex,
  Form,
  Icon,
  Input,
  Status,
} from "@synnaxlabs/pluto";
import { box, location, type record, xy } from "@synnaxlabs/x";
import { memo, type ReactElement, useMemo } from "react";
import { useStore } from "react-redux";

import { selectViewport, useSelectSelected } from "@/arc/selectors";
import { type RootState } from "@/store";

export interface PropertiesProps {
  layoutKey: string;
}

export const PropertiesControls = memo(
  ({ layoutKey }: PropertiesProps): ReactElement | null => {
    const selected = useSelectSelected(layoutKey);
    const nodes = Arc.useSelectNodes({ key: layoutKey });
    const selectedNodeKeys = useMemo(() => {
      const nodeKeys = new Set(nodes.map((n) => n.key));
      return selected.filter((k) => nodeKeys.has(k));
    }, [nodes, selected]);

    if (selected.length === 0)
      return (
        <Status.Summary center variant="disabled" hideIcon>
          Select an Arc element to configure its properties.
        </Status.Summary>
      );
    if (selected.length > 1)
      return (
        <MultiElementProperties
          layoutKey={layoutKey}
          selectedNodeKeys={selectedNodeKeys}
        />
      );
    if (selectedNodeKeys.length === 0) return null;
    return (
      <IndividualProperties
        key={selectedNodeKeys[0]}
        layoutKey={layoutKey}
        nodeKey={selectedNodeKeys[0]}
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
  const config = Arc.useSelectNodeConfig({ key: layoutKey, nodeKey });
  const { dispatch } = Arc.useDispatch();
  const formMethods = Form.use({
    values: structuredClone(config ?? {}),
    sync: true,
    onChange: ({ values }) =>
      dispatch({
        key: layoutKey,
        actions: [
          arc.setNodeConfig({ key: nodeKey, config: values as record.Unknown }),
        ],
      }),
  });
  if (config == null) return null;
  const C = Arc.Graph.Node.REGISTRY[config.type];
  if (C == null) return null;
  return (
    <Flex.Box style={{ height: "100%", padding: "2rem" }} y>
      <Form.Form {...formMethods}>
        <C.Form {...formMethods} key={nodeKey} />
      </Form.Form>
    </Flex.Box>
  );
};

interface MultiElementPropertiesProps {
  layoutKey: string;
  selectedNodeKeys: string[];
}

const MultiElementProperties = ({
  layoutKey,
  selectedNodeKeys,
}: MultiElementPropertiesProps): ReactElement => {
  const { dispatch } = Arc.useDispatch();
  const nodes = Arc.useSelectNodes({ key: layoutKey });
  const store = useStore<RootState>();

  const getLayouts = () => {
    const viewport = selectViewport(store.getState(), layoutKey);
    return selectedNodeKeys
      .map((key) => {
        const node = nodes.find((n) => n.key === key);
        if (node == null) return null;
        try {
          const nodeEl = Diagram.selectNode(key);
          const nodeBox = box.construct(node.position, box.dims(box.construct(nodeEl)));
          const handleEls = nodeEl.getElementsByClassName("react-flow__handle");
          const nodeElBox = box.construct(nodeEl);
          const handles = Array.from(handleEls).map((el) => {
            const pos = box.center(box.construct(el));
            const dist = xy.scale(
              xy.translation(box.topLeft(nodeElBox), pos),
              1 / viewport.zoom,
            );
            const match = el.className.match(/react-flow__handle-(\w+)/);
            if (match == null)
              throw new Error(`[arc] - cannot find handle orientation`);
            const orientation = location.construct(match[1]) as location.Outer;
            return new Diagram.HandleLayout(dist, orientation);
          });
          return new Diagram.NodeLayout(key, nodeBox, handles);
        } catch (e) {
          console.error(e);
        }
        return null;
      })
      .filter((el) => el !== null);
  };

  const align = (direction: "x" | "y"): void => {
    const newPositions = Diagram.alignNodesAlongDirection(getLayouts(), direction);
    dispatch({
      key: layoutKey,
      actions: newPositions.map((n) =>
        arc.setNodePosition({ key: n.key, position: box.topLeft(n.box) }),
      ),
    });
  };

  return (
    <Flex.Box align="start" x style={{ padding: "2rem" }}>
      <Input.Item label="Align">
        <Flex.Box x>
          <Button.Button tooltip="Align nodes vertically" onClick={() => align("x")}>
            <Icon.Align.YCenter />
          </Button.Button>
          <Button.Button tooltip="Align nodes horizontally" onClick={() => align("y")}>
            <Icon.Align.XCenter />
          </Button.Button>
        </Flex.Box>
      </Input.Item>
    </Flex.Box>
  );
};
