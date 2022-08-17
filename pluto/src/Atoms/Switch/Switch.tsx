import {HTMLAttributes} from "react";
import {classList} from "../../util/css";
import "./Switch.css";

export interface SwitchProps extends HTMLAttributes<HTMLInputElement> {

}

const Switch = ({className, ...props}: SwitchProps) => {
    return (
        <label className={classList("pluto-switch__container", className)} >
            <input className="pluto-switch__input" type="checkbox" {...props} />
            <span className="pluto-switch__slider"></span>
        </label>
    )
}

export default Switch;