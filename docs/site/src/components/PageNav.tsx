// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Tree, TreeLeaf } from "@synnaxlabs/pluto";

import { pages } from "@/pages/nav";

export type PageNavLeaf = TreeLeaf;

export interface TOCProps {
  currentPage: string;
}

export const PageNav = ({ currentPage }: TOCProps): JSX.Element => (
  <Tree data={pages} value={[currentPage]} />
);
