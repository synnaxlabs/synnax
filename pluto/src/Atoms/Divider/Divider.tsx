import { Direction } from "../../util/spatial";
import "./Divider.css";

export interface DividerProps {
  direction: Direction;
}

export default function Divider(props: DividerProps) {
  return (
    <div className={`pluto-divider pluto-divider--${props.direction}`}></div>
  );
}
