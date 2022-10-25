import { OntologyID, OntologyResource, OntologyRoot } from "@synnaxlabs/client";
import { Tree, TreeEntry, Header, Space } from "@synnaxlabs/pluto";
import { useEffect, useState } from "react";
import { AiFillFolder } from "react-icons/ai";
import { useActiveClient } from "../cluster/components/useActiveClient";
import { resourceTypes } from "./resources";
import { useDispatch } from "react-redux";
import { Dispatch } from "redux";

const updateTreeEntry = (
  data: TreeEntry[],
  newEntry: Partial<TreeEntry>,
  key: string
) => {
  data.forEach((entry, i) => {
    if (entry.key === key) {
      entry.children = entry.children ?? [];
      data[i] = { ...entry, ...newEntry };
    } else if (entry.children) {
      updateTreeEntry(entry.children, newEntry, key);
    }
  });
};

const convertOntologyResources = (
  dispatch: Dispatch<any>,
  resources: OntologyResource[]
): TreeEntry[] => {
  return resources.map(({ id, entity: { name } }) => {
    const { icon, hasChildren } = resourceTypes(dispatch)[id.type];
    return {
      key: id.toString(),
      title: name,
      icon,
      hasChildren,
      children: [],
    };
  });
};

function ResourcesTree() {
  const client = useActiveClient();
  const [data, setData] = useState<TreeEntry[]>([]);
  const dispatch = useDispatch();

  useEffect(() => {
    if (!client) return;
    const fn = async () => {
      const resources = await client.ontology.retrieveChildren(OntologyRoot);
      console.log(resources);
      setData(convertOntologyResources(dispatch, resources));
    };
    fn();
  }, [client]);

  return (
    <Space empty style={{ height: "100%" }}>
      <Header level="h4" divided icon={<AiFillFolder />}>
        Resources
      </Header>
      <Tree
        data={data}
        style={{ overflowY: "auto", overflowX: "hidden", flexGrow: 1 }}
        onSelect={([key]: string[]) => {
          const id = OntologyID.parseString(key);
          const { onSelect } = resourceTypes(dispatch)[id.type];
          onSelect?.(id);
        }}
        onExpand={(key) => {
          if (!client) return;
          const fn = async () => {
            const resources = await client.ontology.retrieveChildren(
              OntologyID.parseString(key)
            );
            updateTreeEntry(
              data,
              {
                children: convertOntologyResources(dispatch, resources),
              },
              key
            );
            setData(data.map((d) => ({ ...d })));
          };
          fn();
        }}
      />
    </Space>
  );
}

const ResourcesItem = {
  key: "resources",
  icon: <AiFillFolder />,
  content: <ResourcesTree />,
};

export default ResourcesItem;
