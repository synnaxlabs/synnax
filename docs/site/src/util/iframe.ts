const updateHref = () => {
    const url = window.location.href;
    const path = url
      .replace(window.location.origin, "")
      .split("?")[0]
      .split("#")[0];
    const maybeHeading = url.split("#");
    let heading = "";
    if (maybeHeading.length > 1) {
      heading = maybeHeading[1].split("&")[0];
    }
    window.parent.postMessage({ path, heading }, "*");
  };


export const startUpdatingIframeHref = (): void => {
    window.addEventListener("popstate", updateHref);
    updateHref();
}