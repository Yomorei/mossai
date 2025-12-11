export function formatDate(value) {
  if (!value) return "–";
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return value;
  return d.toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "2-digit",
  });
}

export function escapeHtml(str) {
  return String(str)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

export function escapeAttribute(str) {
  return escapeHtml(str).replace(/'/g, "&#39;");
}

export function getServerIdFromPath() {
  const path = window.location.pathname.replace(/\/+$/, "");
  const parts = path.split("/");
  const last = parts[parts.length - 1];
  return last || null;
}
