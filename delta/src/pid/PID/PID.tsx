// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback, useState } from "react";

import { PID as PPID, PIDProps, PIDElementProps } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { ELEMENTS } from "../elements";
import { useSelectPID, useSelectPIDElementProps } from "../store/selectors";
import { setPIDEdges, setPIDElementProps, setPIDNodes } from "../store/slice";

import { LayoutRenderer } from "@/layout";

const PIDElementRenderer = ({
  elementKey,
  position,
  selected,
  layoutKey,
}: PIDElementProps & { layoutKey: string }): ReactElement => {
  const {
    props: { type, ...props },
  } = useSelectPIDElementProps(layoutKey, elementKey);
  const dispatch = useDispatch();

  const handleChange = useCallback(
    (props: object) => {
      dispatch(
        setPIDElementProps({ layoutKey, key: elementKey, props: { type, ...props } })
      );
    },
    [dispatch, elementKey, layoutKey, type]
  );

  const C = ELEMENTS[type];

  return (
    <C.Element
      position={position}
      selected={selected}
      onChange={handleChange}
      editable={true}
      {...props}
    />
  );
};

export const PID: LayoutRenderer = ({ layoutKey }) => {
  const vis = useSelectPID(layoutKey);
  const dispatch = useDispatch();

  const handleEdgesChange: PIDProps["onEdgesChange"] = (cbk) => {
    dispatch(setPIDEdges({ layoutKey, edges: cbk(vis.edges) }));
  };

  const handleNodesChange: PIDProps["onNodesChange"] = (cbk) => {
    dispatch(setPIDNodes({ layoutKey, nodes: cbk(vis.nodes) }));
  };

  const [editable, handleEditableChange] = useState(vis.editable);

  const pidElementRenderer = useCallback(
    (props: PIDElementProps) => {
      return <PIDElementRenderer layoutKey={layoutKey} {...props} />;
    },
    [layoutKey]
  );

  return (
    <PPID
      edges={vis.edges}
      nodes={vis.nodes}
      onEdgesChange={handleEdgesChange}
      onNodesChange={handleNodesChange}
      onEditableChange={handleEditableChange}
      editable={editable}
    >
      {pidElementRenderer}
    </PPID>
  );
};
