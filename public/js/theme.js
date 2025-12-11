export function initTheme() {
  const root = document.documentElement;
  const stored = window.localStorage.getItem("mossai-theme");

  if (stored === "light" || stored === "dark") {
    applyTheme(stored);
  } else {
    const prefersLight = window.matchMedia(
      "(prefers-color-scheme: light)"
    ).matches;
    applyTheme(prefersLight ? "light" : "dark");
  }

  const btn = document.getElementById("theme-toggle");
  if (!btn) return;

  btn.addEventListener("click", () => {
    const current = root.dataset.theme || "dark";
    const next = current === "dark" ? "light" : "dark";
    window.localStorage.setItem("mossai-theme", next);
    applyTheme(next);
  });
}

export function applyTheme(theme) {
  const root = document.documentElement;

  if (theme === "light" || theme === "dark") {
    root.dataset.theme = theme;
  } else {
    root.removeAttribute("data-theme");
    theme = "dark";
  }

  const btn = document.getElementById("theme-toggle");
  if (btn) {
    btn.setAttribute("data-mode", theme);
  }
}
