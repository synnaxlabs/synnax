import { Direction } from "@/util/spatial";
import "./Divider.css";

/** The props for the {@link Divider} component. */
export interface DividerProps {
  direction: Direction;
}

/**
 * Divider renders a vertical or horizontal divided to separate content.
 *
 * @param props - The props for the component.
 * @param props.direction - The directio to render the divider in. Can be "horiztonal"
 * or "vertical".
 */
export const Divider = ({ direction }: DividerProps): JSX.Element => {
  return <div className={`pluto-divider pluto-divider--${direction}`}></div>;
};
