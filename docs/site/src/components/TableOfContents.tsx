import { Tree } from "@synnaxlabs/pluto";
import type { TreeLeaf } from "@synnaxlabs/pluto";

export interface Page {
  key: string;
}

export type TOCLeaf = TreeLeaf<Page>;

export interface TOCProps {
  data: TOCLeaf[];
  currentPage: string;
}

export const TOC = ({ data, currentPage }: TOCProps): JSX.Element => (
  <Tree<Page> data={data} value={[currentPage]} />
);
