export interface UseTreeProps {
  root: ontology.ID;
}

export interface UseTreeDataReturn {
  nodes: Tree.Node[];
  onExpand: (props: Tree.HandleExpandProps<string>) => void;
  setNodes: state.Setter<Tree.Node[]>;
  getResources: (ids: ontology.ID[]) => ontology.Resource[];
}

export const useTreeData = ({ root }: UseTreeProps): UseTreeDataReturn => {};
