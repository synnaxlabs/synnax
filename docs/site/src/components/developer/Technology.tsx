// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import React from "react";
import { Space } from "@synnaxlabs/pluto";

export type TechnologyProps = {
  imageSrc: string;
  imageAlt: string;
  imageWidth: string;
  imageHref: string;
  text: string;
  title: string;
};

export default function Technology(props: TechnologyProps) {
  return (
    <Space size="small">
      <h3>{props.title}</h3>
      <Space direction="horizontal" align="center" size="large">
        <a href={props.imageHref}>
          <img src={props.imageSrc} alt={props.imageAlt} width={props.imageWidth} />
        </a>
        <p>{props.text}</p>
      </Space>
    </Space>
  );
}
