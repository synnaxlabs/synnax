import { type ontology, workspace } from "@synnaxlabs/client";
import { imex } from "@synnaxlabs/pluto";
import { useEffect } from "react";
import { useDispatch } from "react-redux";

import { useSelectPendingUpload } from "@/schematic/selectors";
import { clearPendingUpload } from "@/schematic/slice";
import { Workspace } from "@/workspace";

export const useAutoUpload = (key: string, name: string) => {
  const pendingUpload = useSelectPendingUpload(key);
  const { update: import_ } = imex.useImport();
  const workspaceKey = Workspace.useSelectActiveKey();
  const dispatch = useDispatch();
  useEffect(() => {
    if (pendingUpload == null) return;
    const parent: ontology.ID | null =
      workspaceKey != null ? workspace.ontologyID(workspaceKey) : null;
    const envelope = { ...pendingUpload, type: "schematic", key, name };
    import_({ parent, envelopes: envelope });
    dispatch(clearPendingUpload({ key }));
  }, [pendingUpload, workspaceKey, key, name, dispatch]);
};
