import { useDispatch } from "react-redux";
interface HandleLinkProps {
  url: string;
  placer: Layout.Placer;
  client: Synnax;
  addStatus: (status: Status.CrudeSpec) => void;
}

export type Handler = (props: HandleLinkProps) => boolean;

export const useDeep = () => {
  const dispatch = useDispatch();
};
