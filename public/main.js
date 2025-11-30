const serverGridEl = document.getElementById("server-grid");
const loadingEl = document.getElementById("leaderboard-loading");
const errorEl = document.getElementById("leaderboard-error");
const wrapperEl = document.getElementById("leaderboard-wrapper");

const FALLBACK_SERVERS = [
  {
    id: 1,
    server_name: "M1PPosu",
    owner: "Yomorei",
    votes: 67,
    added: new Date("2025-11-30").toISOString(),
    url: "https://m1pposu.dev",
    logo: "/static/m1pplogo.png",
    online: 6,
    registered: 993,
    tags: ["Relax", "Autopilot", "Catch", "Mania", "Taiko"],
    description:
      "a osu! server where we rank the unrankable. Map packs, goofy plays, and more.",
    status: "online",
    status_text: "6 players online",
  },
];

const SKELETON_CARD_COUNT = 10;

function showLoading() {
  loadingEl.classList.remove("hidden");
  errorEl.classList.add("hidden");
  wrapperEl.classList.add("hidden");
}

async function fetchLeaderboard() {
  showLoading();

  try {
    const res = await fetch("/leaderboard", {
      headers: { Accept: "application/json" },
    });

    if (!res.ok) {
      throw new Error(`HTTP ${res.status}`);
    }

    const data = await res.json();
    const dbServers = Array.isArray(data) ? data : [];
    const serversToRender =
      dbServers.length > 0 ? dbServers : FALLBACK_SERVERS;

    renderLeaderboard(serversToRender);

    loadingEl.classList.add("hidden");
    wrapperEl.classList.remove("hidden");
  } catch (err) {
    loadingEl.classList.add("hidden");
    errorEl.classList.remove("hidden");
    console.error("Failed to load leaderboard", err);
  }
}

function renderLeaderboard(servers) {
  serverGridEl.innerHTML = "";

  servers.forEach((rawServer, index) => {
    const server = { rank: index + 1, ...rawServer };
    const card = createServerCard(server);
    serverGridEl.appendChild(card);
  });

  for (let i = 0; i < SKELETON_CARD_COUNT; i++) {
    const skeletonCard = createSkeletonCard();
    serverGridEl.appendChild(skeletonCard);
  }
}

function createServerCard(server) {
  const card = document.createElement("article");
  card.className = "server-card";

  const logoUrl = server.logo_url || server.logo || "";
  const tags = Array.isArray(server.tags) ? server.tags : [];
  const votes = server.votes ?? 0;
  const addedFormatted = formatDate(server.added);
  const name = server.server_name || "";
  const owner = server.owner || "Unknown";
  const description = server.description || server.tagline || "";
  const statusIsOnline = server.status === "online";
  const statusClass = statusIsOnline ? "status-online" : "status-offline";
  const statusText = server.status_text || "0 players online";
  const rankLabel = server.rank ?? 0;

  card.innerHTML = `
    <header class="server-card-header">
      <span class="server-rank-badge">#${rankLabel}</span>
    </header>

    <div class="server-main-row">
      <div class="server-logo">
        ${
          logoUrl
            ? `<img src="${escapeAttribute(logoUrl)}" alt="${escapeAttribute(
                name
              )} logo" />`
            : `<span class="server-logo-fallback">${escapeHtml(
                getInitials(name)
              )}</span>`
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

    <div class="server-added">
      Added: ${escapeHtml(addedFormatted)}
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
      </div>
      <button class="vote-button"
              data-server-name="${escapeAttribute(name)}"
              data-server-id="${escapeAttribute(String(server.id ?? ""))}">
        Vote
      </button>
    </div>
  `;

  return card;
}

function createSkeletonCard() {
  const card = document.createElement("article");
  card.className = "server-card server-card-skeleton";

  card.innerHTML = `
    <div class="skeleton-header">
      <div class="skeleton-block skeleton-rank"></div>
    </div>
    <div class="skeleton-main">
      <div class="skeleton-circle"></div>
      <div class="skeleton-lines">
        <div class="skeleton-block skeleton-line"></div>
        <div class="skeleton-block skeleton-line short"></div>
      </div>
    </div>
    <div class="skeleton-block skeleton-description"></div>
    <div class="skeleton-stats">
      <div class="skeleton-block skeleton-stat"></div>
      <div class="skeleton-block skeleton-stat"></div>
      <div class="skeleton-block skeleton-stat"></div>
    </div>
    <div class="skeleton-actions">
      <div class="skeleton-block skeleton-pill"></div>
      <div class="skeleton-block skeleton-pill"></div>
    </div>
  `;

  return card;
}

/* helpers */

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

function getInitials(name) {
  if (!name) return "?";
  const parts = String(name).trim().split(/\s+/);
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[1][0]).toUpperCase();
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

/* vote handling */

async function handleVoteClick(button) {
  const serverId = button.getAttribute("data-server-id");
  const serverName = button.getAttribute("data-server-name") || "this server";
  if (!serverId) return;

  button.disabled = true;

  try {
    const res = await fetch(`/server/${encodeURIComponent(serverId)}/vote`, {
      method: "POST",
      headers: {
        Accept: "text/plain,application/json",
      },
    });

    if (!res.ok) {
      throw new Error(`HTTP ${res.status}`);
    }

    const card = button.closest(".server-card");
    const votesEl = card && card.querySelector("[data-votes]");
    if (votesEl) {
      const current = parseInt(votesEl.textContent || "0", 10) || 0;
      votesEl.textContent = String(current + 1);
    }
  } catch (err) {
    console.error("Failed to send vote", err);
    alert(`Couldn't vote for ${serverName}. Please try again.`);
  } finally {
    button.disabled = false;
  }
}

/* events */

document.addEventListener("click", (event) => {
  const target = event.target;
  if (!(target instanceof HTMLElement)) return;

  if (target.matches(".vote-button")) {
    event.preventDefault();
    handleVoteClick(target);
  }
});

document.addEventListener("DOMContentLoaded", () => {
  fetchLeaderboard();
});
