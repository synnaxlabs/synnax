// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

let timeOut: NodeJS.Timeout;

export const addCodeButtonListeners = (): void => {
  const listeners: { button: Element; listener: () => void }[] = [];

  const handleCopy = (button: Element) => {
    const code = button.parentElement?.querySelector("code");
    if (code == null) return;
    navigator.clipboard
      .writeText(code.innerText)
      .then(() => {
        const copy = button.querySelector(".copy") as HTMLElement;
        const check = button.querySelector(".check") as HTMLElement;
        if (copy == null || check == null) return;
        copy.style.display = "none";
        check.style.display = "block";
        clearTimeout(timeOut);
        timeOut = setTimeout(() => {
          copy.style.display = "block";
          check.style.display = "none";
        }, 1000);
      })
      .catch(console.error);
  };

  setInterval(() => {
    const buttons = document.querySelectorAll(".astro-code-wrapper button");
    listeners.forEach(({ button, listener }) =>
      button.removeEventListener("click", listener),
    );
    buttons.forEach((button) => {
      const listener = () => handleCopy(button);
      button.addEventListener("click", listener);
      listeners.push({ button, listener });
    });
  }, 1000);
};
