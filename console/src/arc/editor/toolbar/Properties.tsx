// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
import { box, location, xy } from "@synnaxlabs/x";
import { memo, type ReactElement } from "react";
import { useDispatch, useStore } from "react-redux";

import {
  selectViewport,
  useSelectRequiredNodeProps,
  useSelectSelectedElementDigests,
  useSelectSelectedElementsProps,
} from "@/arc/selectors";
import { setElementProps, setNodePositions } from "@/arc/slice";
import { type RootState } from "@/store";

export interface PropertiesProps {
  layoutKey: string;
}

export const PropertiesControls = memo(
  ({ layoutKey }: PropertiesProps): ReactElement | null => {
    const digests = useSelectSelectedElementDigests(layoutKey);
    if (digests.length === 0)
      return (
        <Status.Summary center variant="disabled" hideIcon>
          Select a arc element to configure its properties.
        </Status.Summary>
      );

    if (digests.length > 1) return <MultiElementProperties layoutKey={layoutKey} />;

    const selected = digests[0];

    if (selected.type === "edge") return null;
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
  const C = Arc.Stage.REGISTRY[props.key];
  const dispatch = useDispatch();

  const onChange = (key: string, props: any): void => {
    dispatch(setElementProps({ layoutKey, key, props }));
  };

  const formMethods = Form.use({
    values: structuredClone(props),
    sync: true,
    onChange: ({ values }) => onChange(nodeKey, values),
  });

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
}

const MultiElementProperties = ({
  layoutKey,
}: MultiElementPropertiesProps): ReactElement => {
  const elements = useSelectSelectedElementsProps(layoutKey);
  const dispatch = useDispatch();

  const store = useStore<RootState>();

  const getLayouts = () => {
    const viewport = selectViewport(store.getState(), layoutKey);
    return elements
      .map((el) => {
        if (el.type !== "node") return null;
        // grab all child elements with the class 'react-flow__handle'
        try {
          const nodeEl = Diagram.selectNode(el.key);
          const nodeBox = box.construct(
            el.node.position,
            box.dims(box.construct(nodeEl)),
          );
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
          return new Diagram.NodeLayout(el.key, nodeBox, handles);
        } catch (e) {
          console.error(e);
        }
        return null;
      })
      .filter((el) => el !== null);
  };

  return (
    <Flex.Box align="start" x style={{ padding: "2rem" }}>
      <Input.Item label="Align">
        <Flex.Box x>
          <Button.Button
            tooltip="Align nodes vertically"
            onClick={() => {
              const newPositions = Diagram.alignNodesAlongDirection(getLayouts(), "x");
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
              const newPositions = Diagram.alignNodesAlongDirection(getLayouts(), "y");
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
        </Flex.Box>
      </Input.Item>
    </Flex.Box>
  );
};
