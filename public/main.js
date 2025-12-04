const serverGridEl = document.getElementById("server-grid");
const loadingEl = document.getElementById("leaderboard-loading");
const errorEl = document.getElementById("leaderboard-error");
const wrapperEl = document.getElementById("leaderboard-wrapper");

function showLoading() {
  if (!loadingEl || !errorEl || !wrapperEl) return;
  loadingEl.classList.remove("hidden");
  errorEl.classList.add("hidden");
  wrapperEl.classList.add("hidden");
}

async function fetchLeaderboard() {
  if (!serverGridEl) return;

  showLoading();

  try {
    const res = await fetch("/leaderboard", {
      headers: { Accept: "application/json" },
    });

    if (!res.ok) {
      throw new Error("HTTP " + res.status);
    }

    const data = await res.json();
    const servers = Array.isArray(data) ? data : [];

    renderLeaderboard(servers);

    if (loadingEl) loadingEl.classList.add("hidden");
    if (wrapperEl) wrapperEl.classList.remove("hidden");
  } catch (err) {
    console.error("Failed to load leaderboard", err);
    if (loadingEl) loadingEl.classList.add("hidden");
    if (errorEl) errorEl.classList.remove("hidden");
  }
}

function renderLeaderboard(servers) {
  serverGridEl.innerHTML = "";

  if (!servers.length) {
    const empty = document.createElement("div");
    empty.className = "notice";
    empty.textContent = "No servers are listed yet.";
    serverGridEl.appendChild(empty);
    return;
  }

  servers.forEach((rawServer, index) => {
    const server = { rank: index + 1, ...rawServer };
    const card = createServerCard(server);
    serverGridEl.appendChild(card);
  });
}

function createServerCard(server) {
  const card = document.createElement("article");
  card.className = "server-card";

  const rawName = server.server_name || server.name || "";
  const name = rawName;
  const normalizedName = String(rawName).trim().toLowerCase();

  let logoUrl = server.logo_url || server.logo || "";
  if (!logoUrl && normalizedName === "m1pposu") {
    logoUrl = "/static/m1pplogo.png";
  }

  const tagsRaw = server.tags || "";
  const tags =
    typeof tagsRaw === "string"
      ? tagsRaw
          .split(",")
          .map((t) => t.trim())
          .filter(Boolean)
      : Array.isArray(tagsRaw)
      ? tagsRaw
      : [];

  const votes = server.votes ?? 0;
  const addedFormatted = formatDate(server.added);
  const owner = server.owner || "Unknown";
  const description = server.description || server.tagline || "";

  const statusIsOnline = server.status === "online";
  const statusClass = statusIsOnline ? "status-online" : "status-offline";
  const statusText = server.status_text || "0 players online";
  const rankLabel = server.rank ?? 0;
  const id = server.id ?? "";

  let rankClass = "";
  if (rankLabel === 1) {
    rankClass = "server-rank-1";
  } else if (rankLabel === 2) {
    rankClass = "server-rank-2";
  } else if (rankLabel === 3) {
    rankClass = "server-rank-3";
  }

  card.innerHTML = `
    <header class="server-card-header">
      <span class="server-rank-badge ${rankClass}">#${rankLabel}</span>
      <span class="server-added">${addedFormatted}</span>
    </header>

    <div class="server-main-row">
      <div class="server-logo">
        ${
          logoUrl
            ? `<img src="${escapeAttribute(logoUrl)}" alt="${escapeAttribute(
                name
              )} logo" />`
            : ""
        }
      </div>
      <div class="server-main-text">
        <div class="server-name">${escapeHtml(name)}</div>
        <div class="server-status">
          <span class="status-dot ${statusClass}"></span>
          <span class="status-text">${escapeHtml(statusText)}</span>
        </div>
        <div class="server-owner">Owner: ${escapeHtml(owner)}</div>
      </div>
    </div>

    ${
      description
        ? `<p class="server-description">${escapeHtml(description)}</p>`
        : ""
    }

    <div class="server-metrics-row">
      <div class="metric">
        <div class="metric-label">Online</div>
        <div class="metric-value">${server.online ?? "–"}</div>
      </div>
      <div class="metric">
        <div class="metric-label">Registered</div>
        <div class="metric-value">${server.registered ?? "–"}</div>
      </div>
      <div class="metric">
        <div class="metric-label">Votes</div>
        <div class="metric-value" data-votes>${votes}</div>
      </div>
    </div>

    ${
      tags.length
        ? `<div class="server-tags-row">
            ${tags
              .map(
                (tag) =>
                  `<span class="tag-pill">${escapeHtml(String(tag))}</span>`
              )
              .join("")}
          </div>`
        : ""
    }

    <div class="server-actions-row">
      <div class="server-actions-left">
        ${
          server.url
            ? `<a href="${escapeAttribute(
                server.url
              )}" target="_blank" rel="noopener noreferrer" class="badge-link">
                 Website
               </a>`
            : ""
        }
        ${
          id
            ? `<a href="/servers/${encodeURIComponent(
                String(id)
              )}" class="badge-link">
                 Details
               </a>`
            : ""
        }
      </div>
      <button class="vote-button"
              data-server-name="${escapeAttribute(name)}"
              data-server-id="${escapeAttribute(String(id))}">
        Vote
      </button>
    </div>
  `;

  return card;
}

function formatDate(value) {
  if (!value) return "–";
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return value;
  return d.toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "2-digit",
  });
}

function escapeHtml(str) {
  return String(str)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function escapeAttribute(str) {
  return escapeHtml(str).replace(/'/g, "&#39;");
}

async function handleVoteClick(button) {
  const serverId = button.getAttribute("data-server-id");
  const serverName = button.getAttribute("data-server-name") || "this server";
  if (!serverId) return;

  const nameInput = window.prompt(
    `Enter your in-game name to vote for ${serverName}:`
  );

  if (nameInput == null) {
    return;
  }

  const trimmed = nameInput.trim();
  if (!trimmed) {
    return;
  }

  button.disabled = true;

  try {
    const res = await fetch(`/server/${encodeURIComponent(serverId)}/vote`, {
      method: "POST",
      headers: {
        Accept: "text/plain,application/json",
        "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8",
      },
      body: new URLSearchParams({ name: trimmed }).toString(),
    });

    const text = await res.text();

    if (!res.ok) {
      alert(text || `Couldn't vote for ${serverName}.`);
      return;
    }

    const card = button.closest(".server-card");
    const votesEl = card && card.querySelector("[data-votes]");
    if (votesEl) {
      const current = parseInt(votesEl.textContent || "0", 10) || 0;
      votesEl.textContent = String(current + 1);
    }

    const detailVotesEl = document.getElementById("server-detail-votes");
    if (detailVotesEl) {
      const currentDetail =
        parseInt(detailVotesEl.textContent || "0", 10) || 0;
      detailVotesEl.textContent = String(currentDetail + 1);
    }
  } catch (err) {
    console.error("Failed to send vote", err);
    alert(`Couldn't vote for ${serverName}. Please try again.`);
  } finally {
    button.disabled = false;
  }
}

function getServerIdFromPath() {
  const path = window.location.pathname.replace(/\/+$/, "");
  const parts = path.split("/");
  const last = parts[parts.length - 1];
  return last || null;
}

async function initServerDetail() {
  const root = document.getElementById("server-detail");
  if (!root) return;

  const loading = document.getElementById("server-detail-loading");
  const error = document.getElementById("server-detail-error");
  const body = document.getElementById("server-detail-body");

  const id = getServerIdFromPath();
  if (!id) {
    if (loading) loading.classList.add("hidden");
    if (error) {
      error.textContent = "Invalid server URL.";
      error.classList.remove("hidden");
    }
    return;
  }

  if (loading) loading.classList.remove("hidden");
  if (error) error.classList.add("hidden");
  if (body) body.classList.add("hidden");

  try {
    const res = await fetch(`/server/${encodeURIComponent(id)}`, {
      headers: { Accept: "application/json" },
    });

    if (!res.ok) {
      const text = await res.text();
      if (loading) loading.classList.add("hidden");
      if (error) {
        error.textContent = text || "Couldn't load this server.";
        error.classList.remove("hidden");
      }
      return;
    }

    const data = await res.json();
    renderServerDetail(data);

    if (loading) loading.classList.add("hidden");
    if (body) body.classList.remove("hidden");
  } catch (err) {
    console.error("Failed to load server detail", err);
    if (loading) loading.classList.add("hidden");
    if (error) {
      error.textContent = "Couldn't load this server.";
      error.classList.remove("hidden");
    }
  }
}

function renderServerDetail(server) {
  const name = server.server_name || server.name || "";
  const owner = server.owner || "Unknown";
  const description = server.description || "";
  const votes = server.votes ?? 0;
  const added = server.added || "";
  const addedFormatted = formatDate(added);
  const id = server.id ?? "";
  const url = server.url || "";

  const nameEl = document.getElementById("server-detail-name");
  const ownerEl = document.getElementById("server-detail-owner");
  const ownerShortEl = document.getElementById("server-detail-owner-short");
  const addedEl = document.getElementById("server-detail-added");
  const listedEl = document.getElementById("server-detail-listed");
  const descEl = document.getElementById("server-detail-description");
  const votesEl = document.getElementById("server-detail-votes");
  const logoEl = document.getElementById("server-detail-logo");
  const websiteEl = document.getElementById("server-detail-website");
  const voteButton = document.getElementById("server-detail-vote");

  if (nameEl) nameEl.textContent = name;
  if (ownerEl) ownerEl.textContent = "Owner: " + owner;
  if (ownerShortEl) ownerShortEl.textContent = owner;
  if (addedEl) addedEl.textContent = "Added " + addedFormatted;
  if (listedEl) listedEl.textContent = addedFormatted;
  if (votesEl) votesEl.textContent = String(votes);
  if (descEl) {
    if (description) {
      descEl.textContent = description;
    } else {
      descEl.textContent = "No description has been provided yet.";
    }
  }

  if (logoEl) {
    let logoUrl = server.logo_url || server.logo || "";
    const normalizedName = String(name).trim().toLowerCase();
    if (!logoUrl && normalizedName === "m1pposu") {
      logoUrl = "/static/m1pplogo.png";
    }
    if (logoUrl) {
      logoEl.innerHTML = `<img src="${escapeAttribute(
        logoUrl
      )}" alt="${escapeAttribute(name)} logo" />`;
    } else {
      logoEl.textContent = "";
    }
  }

  if (websiteEl) {
    if (url) {
      websiteEl.href = url;
      websiteEl.classList.remove("hidden");
    } else {
      websiteEl.href = "#";
      websiteEl.classList.add("hidden");
    }
  }

  if (voteButton) {
    voteButton.setAttribute("data-server-id", String(id));
    voteButton.setAttribute("data-server-name", name || "this server");
  }

  const backBtn = document.getElementById("server-detail-back");
  if (backBtn) {
    backBtn.addEventListener("click", () => {
      window.location.href = "/";
    });
  }
}

/* list page: tags, TOS, success banner */

function initListPage() {
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

/* global event wiring */

document.addEventListener("click", (event) => {
  const target = event.target;
  if (!(target instanceof HTMLElement)) return;

  if (target.matches(".vote-button")) {
    event.preventDefault();
    handleVoteClick(target);
  }
});

function initAuth() {
  const loginBtn = document.getElementById("discord-login");
  const userContainer = document.getElementById("nav-user");
  const avatarEl = document.getElementById("nav-user-avatar");
  const triggerEl = document.getElementById("nav-user-trigger");
  const menuEl = document.getElementById("nav-user-menu");
  const logoutBtn = document.getElementById("nav-logout");

  if (userContainer) {
    userContainer.classList.add("hidden");
  }

  if (loginBtn) {
    loginBtn.classList.remove("hidden");
    loginBtn.addEventListener("click", () => {
      window.location.href = "/auth/discord/login";
    });
  }

  if (!userContainer) return;

  const closeMenu = () => {
    userContainer.classList.remove("nav-user-open");
    if (triggerEl) {
      triggerEl.setAttribute("aria-expanded", "false");
    }
  };

  const openMenu = () => {
    userContainer.classList.add("nav-user-open");
    if (triggerEl) {
      triggerEl.setAttribute("aria-expanded", "true");
    }
  };

  fetch("/auth/me", { headers: { Accept: "application/json" } })
    .then((res) => res.json())
    .then((data) => {
      if (!data || !data.authenticated) {
        userContainer.classList.add("hidden");
        if (loginBtn) loginBtn.classList.remove("hidden");
        return;
      }

      if (loginBtn) loginBtn.classList.add("hidden");
      userContainer.classList.remove("hidden");

      if (avatarEl && data.avatar_url) {
        avatarEl.src = data.avatar_url;
        avatarEl.alt = data.username || "Discord avatar";
      }

      if (triggerEl && menuEl) {
        triggerEl.addEventListener("click", (e) => {
          e.stopPropagation();
          const isOpen = userContainer.classList.contains("nav-user-open");
          if (isOpen) {
            closeMenu();
          } else {
            openMenu();
          }
        });

        document.addEventListener("click", (event) => {
          const target = event.target;
          if (!(target instanceof HTMLElement)) return;
          if (!userContainer.contains(target)) {
            closeMenu();
          }
        });
      }
    })
    .catch(() => {
      userContainer.classList.add("hidden");
      if (loginBtn) loginBtn.classList.remove("hidden");
    });

  if (logoutBtn) {
    logoutBtn.addEventListener("click", (e) => {
      e.preventDefault();
      fetch("/auth/logout", { method: "POST" }).finally(() => {
        window.location.reload();
      });
    });
  }
}


document.addEventListener("DOMContentLoaded", () => {
  initTheme();
  initAuth();

  const detailRoot = document.getElementById("server-detail");
  const listForm = document.getElementById("server-request-form");

  if (detailRoot) {
    initServerDetail();
  } else if (listForm) {
    initListPage();
  } else if (serverGridEl) {
    fetchLeaderboard();
  }
});


/* theme */

function initTheme() {
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

function applyTheme(theme) {
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
