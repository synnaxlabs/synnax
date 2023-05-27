import { CanvasHTMLAttributes, DetailedHTMLProps } from "react";

type HTMLCanvasProps = DetailedHTMLProps<
  CanvasHTMLAttributes<HTMLCanvasElement>,
  HTMLCanvasElement
>;

export interface CanvasProps extends Omit<HTMLCanvasProps, "ref"> {}

export const Canvas = () => {};
