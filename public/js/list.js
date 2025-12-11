import { escapeHtml } from "./dom-utils.js";

export function initListPage() {
  const successEl = document.getElementById("list-success");
  const formEl = document.getElementById("server-request-form");

  if (successEl) {
    const params = new URLSearchParams(window.location.search);
    if (params.get("submitted") === "1") {
      successEl.classList.remove("hidden");
    } else {
      successEl.classList.add("hidden");
    }
  }

  if (!formEl) return;

  initListTags();
  initListTos();
}

function initListTags() {
  const hidden = document.getElementById("tags");
  const input = document.getElementById("tags-input");
  const chips = document.getElementById("tag-chips");

  if (!hidden || !input || !chips) return;

  let tags = hidden.value
    .split(",")
    .map((t) => t.trim())
    .filter(Boolean);

  const render = () => {
    chips.innerHTML = "";
    tags.forEach((tag, index) => {
      const span = document.createElement("span");
      span.className = "tag-pill";
      span.dataset.index = String(index);
      span.innerHTML = `${escapeHtml(
        tag
      )}<button type="button" class="tag-pill-remove" aria-label="Remove tag">&times;</button>`;
      chips.appendChild(span);
    });
    hidden.value = tags.join(",");
  };

  const addTag = (raw) => {
    const value = raw.trim().toLowerCase();
    if (!value) return;
    if (tags.includes(value)) return;
    tags.push(value);
    render();
  };

  input.addEventListener("keydown", (e) => {
    if (e.key === "Enter" || e.key === ",") {
      e.preventDefault();
      const value = input.value.replace(",", " ");
      addTag(value);
      input.value = "";
    } else if (e.key === "Backspace" && input.value === "") {
      if (tags.length > 0) {
        tags.pop();
        render();
      }
    }
  });

  chips.addEventListener("click", (e) => {
    const target = e.target;
    if (!(target instanceof HTMLElement)) return;
    if (!target.classList.contains("tag-pill-remove")) return;
    const parent = target.closest(".tag-pill");
    if (!parent) return;
    const index = parent.dataset.index;
    if (index == null) return;
    const idx = Number(index);
    if (Number.isNaN(idx)) return;
    tags.splice(idx, 1);
    render();
  });

  render();
}

function initListTos() {
  const box = document.getElementById("tos-box");
  const checkbox = document.getElementById("tos_accept");
  const label = document.getElementById("tos_accept_label");

  if (!box || !checkbox) return;

  const maybeUnlock = () => {
    const atBottom =
      box.scrollTop + box.clientHeight >= box.scrollHeight - 8;
    if (atBottom && checkbox.disabled) {
      checkbox.disabled = false;
      box.classList.add("tos-scrolled");
      if (label) {
        label.classList.add("tos-accept-enabled");
      }
    }
  };

  maybeUnlock();
  box.addEventListener("scroll", maybeUnlock);
}
