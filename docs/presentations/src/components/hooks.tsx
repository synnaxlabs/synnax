import { useEffect, useState } from "react";

export const useArrowKeys = ({
  left,
  right,
}: {
  left?: () => void;
  right?: () => void;
}) => {
  useKeyPress(39, right);
  useKeyPress(37, left);
};

function useKeyPress(targetKey: number, f?: () => void) {
  function downHandler({ keyCode }: { keyCode: number }) {
    if (keyCode === targetKey) {
      if (f) f();
    }
  }
  useEffect(() => {
    window.addEventListener("keydown", downHandler);
    return () => {
      window.removeEventListener("keydown", downHandler);
    };
  }, [f]);
}
