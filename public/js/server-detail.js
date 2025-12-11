import { formatDate, escapeAttribute, getServerIdFromPath } from "./dom-utils.js";

export function initServerDetail() {
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

  initAdminRemoveButton(id);

  if (loading) loading.classList.remove("hidden");
  if (error) error.classList.add("hidden");
  if (body) body.classList.add("hidden");

  fetch(`/server/${encodeURIComponent(id)}`, {
    headers: { Accept: "application/json" },
    credentials: "include",
  })
    .then(async (res) => {
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
    })
    .catch(() => {
      if (loading) loading.classList.add("hidden");
      if (error) {
        error.textContent = "Couldn't load this server.";
        error.classList.remove("hidden");
      }
    });

  const backBtn = document.getElementById("server-detail-back");
  if (backBtn) {
    backBtn.addEventListener("click", () => {
      window.location.href = "/";
    });
  }
}

function renderServerDetail(server) {
  const name = server.server_name || server.name || "";
  const owner = server.owner || server.owner_name || "Unknown";
  const description = server.description || "";
  const votes = server.votes ?? 0;
  const added = server.added || server.created_at || "";
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
    descEl.textContent =
      description || "No description has been provided yet.";
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
}

function initAdminRemoveButton(serverIdFromPath) {
  const removeBtn = document.getElementById("server-remove-btn");
  if (!removeBtn) return;

  const navAdmin = document.getElementById("nav-admin");
  if (!navAdmin) return;

  let attempts = 0;
  const maxAttempts = 40;
  const intervalMs = 100;

  const tryWire = () => {
    attempts += 1;

    const isAdminVisible = !navAdmin.classList.contains("hidden");

    if (isAdminVisible) {
      removeBtn.classList.remove("hidden");

      removeBtn.addEventListener("click", async () => {
        if (
          !confirm(
            "Are you sure you want to remove this server from mossai?"
          )
        ) {
          return;
        }

        const serverId = serverIdFromPath || getServerIdFromPath();
        if (!serverId) {
          alert("Could not determine server id.");
          return;
        }

        try {
          const resp = await fetch(
            `/api/admin/servers/${encodeURIComponent(serverId)}/remove`,
            {
              method: "POST",
              credentials: "include",
              headers: {
                Accept: "application/json",
              },
            }
          );

          if (!resp.ok) {
            const text = await resp.text();
            alert(text || "Failed to remove server.");
            return;
          }

          window.location.href = "/";
        } catch (err) {
          console.error("remove server failed", err);
          alert("Failed to remove server.");
        }
      });

      return; 
    }

    if (attempts < maxAttempts) {
      window.setTimeout(tryWire, intervalMs);
    }
  };

  tryWire();
}
