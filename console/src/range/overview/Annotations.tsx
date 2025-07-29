// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import { Align, Header, Ranger } from "@synnaxlabs/pluto";

import { Annotation } from "@/annotation";

export interface AnnotationsProps {
  rangeKey: string;
}

export const Annotations = ({ rangeKey }: AnnotationsProps) => {
  const range = Ranger.useRetrieve({ params: { key: rangeKey } });
  return (
    <Align.Space y>
      <Header.Header level="h4" bordered={false} borderShade={5} padded>
        <Header.Title shade={11} weight={450}>
          Annotations
        </Header.Title>
      </Header.Header>
      <Annotation.List
        parent={ranger.ontologyID(rangeKey)}
        parentStart={range?.data?.timeRange.start}
      />
    </Align.Space>
  );
};
