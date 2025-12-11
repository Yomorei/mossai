import { formatDate, escapeHtml, escapeAttribute } from "./dom-utils.js";

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

export function initLeaderboard() {
  if (!serverGridEl) return;
  fetchLeaderboard();
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
  } catch (_err) {
    if (loadingEl) loadingEl.classList.add("hidden");
    if (errorEl) errorEl.classList.remove("hidden");
  }
}

function renderLeaderboard(servers) {
  if (!serverGridEl) return;
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
  const owner = server.owner || server.owner_name || "Unknown";
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

export async function handleVoteClick(button) {
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
  } catch (_err) {
    alert(`Couldn't vote for ${serverName}. Please try again.`);
  } finally {
    button.disabled = false;
  }
}
