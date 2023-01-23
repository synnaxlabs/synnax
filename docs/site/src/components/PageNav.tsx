import { Tree } from "@synnaxlabs/pluto";
import type { TreeLeaf } from "@synnaxlabs/pluto";

export type PageNavLeaf = TreeLeaf;

export interface TOCProps {
  data: PageNavLeaf[];
  currentPage: string;
}

export const PageNav = ({ data, currentPage }: TOCProps): JSX.Element => (
  <Tree data={data} value={[currentPage]} />
);
