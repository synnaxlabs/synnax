let timeOut: any = null;
export const addListeners = (): void => {
  const buttons = document.querySelectorAll(".astro-code-wrapper button");
  buttons.forEach((button) => {
    button.addEventListener("click", () => {
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
          clearTimeout(timeOut as string);
          timeOut = setTimeout(() => {
            copy.style.display = "block";
            check.style.display = "none";
          }, 1000);
        })
        .catch(console.error);
    });
  });
};
