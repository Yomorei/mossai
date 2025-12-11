import { escapeHtml } from "./dom-utils.js";

const adminState = {
  requestsById: new Map(),
  editingId: null,
};

export function initAdminRequests() {
  const root = document.getElementById("admin-requests-root");
  if (!root) return;

  const tableBody = document.getElementById("admin-requests-table-body");
  const emptyNotice = document.getElementById("admin-requests-empty");
  const errorNotice = document.getElementById("admin-requests-error");

  const overlay = document.getElementById("admin-edit-overlay");
  const editForm = document.getElementById("admin-edit-form");
  const editError = document.getElementById("admin-edit-error");
  const closeBtn = document.getElementById("admin-edit-close");
  const cancelBtn = document.getElementById("admin-edit-cancel");

  if (
    !tableBody ||
    !emptyNotice ||
    !errorNotice ||
    !overlay ||
    !editForm ||
    !editError ||
    !closeBtn ||
    !cancelBtn
  ) {
    return;
  }

  function openEditModal(req) {
    adminState.editingId = req.id;

    editForm.elements["server_name"].value = req.server_name || "";
    editForm.elements["url"].value = req.url || "";
    editForm.elements["logo_url"].value = req.logo_url || "";
    editForm.elements["description"].value = req.description || "";
    editForm.elements["tags"].value = (req.tags || []).join(", ");
    editForm.elements["owner_name"].value = req.owner_name || "";
    editForm.elements["owner_discord"].value = req.owner_discord || "";

    editError.classList.add("hidden");
    overlay.classList.remove("hidden");
  }

  function closeEditModal() {
    adminState.editingId = null;
    overlay.classList.add("hidden");
  }

  function renderTable(requests) {
    tableBody.innerHTML = "";
    adminState.requestsById.clear();

    if (!requests.length) {
      emptyNotice.classList.remove("hidden");
      return;
    }

    emptyNotice.classList.add("hidden");

    requests.forEach((req) => {
      adminState.requestsById.set(req.id, req);

      const tr = document.createElement("tr");

      const serverTd = document.createElement("td");
      serverTd.innerHTML = `
        <div class="admin-requests-server-name">${escapeHtml(
          req.server_name || ""
        )}</div>
        ${
          req.description
            ? `<div class="admin-requests-description">${escapeHtml(
                req.description
              )}</div>`
            : ""
        }
      `;
      tr.appendChild(serverTd);

      const ownerTd = document.createElement("td");
      ownerTd.textContent = req.owner_name || "";
      tr.appendChild(ownerTd);

      const discordTd = document.createElement("td");
      discordTd.textContent = req.owner_discord || "";
      tr.appendChild(discordTd);

      const tagsTd = document.createElement("td");
      const tagsWrap = document.createElement("div");
      tagsWrap.className = "admin-requests-tags";
      (req.tags || []).forEach((tag) => {
        const span = document.createElement("span");
        span.className = "admin-requests-tag";
        span.textContent = tag;
        tagsWrap.appendChild(span);
      });
      tagsTd.appendChild(tagsWrap);
      tr.appendChild(tagsTd);

      const submittedTd = document.createElement("td");
      submittedTd.textContent = req.submitted_at_human || "";
      tr.appendChild(submittedTd);

      const actionsTd = document.createElement("td");
      const actionsWrap = document.createElement("div");
      actionsWrap.className = "admin-requests-actions";

      const editBtn = document.createElement("button");
      editBtn.type = "button";
      editBtn.className = "admin-requests-btn";
      editBtn.textContent = "Edit";
      editBtn.addEventListener("click", () => openEditModal(req));

      const approveBtn = document.createElement("button");
      approveBtn.type = "button";
      approveBtn.className = "admin-requests-btn admin-requests-btn-approve";
      approveBtn.textContent = "Approve";
      approveBtn.addEventListener("click", () =>
        mutateRequest(req.id, "approve")
      );

      const rejectBtn = document.createElement("button");
      rejectBtn.type = "button";
      rejectBtn.className = "admin-requests-btn admin-requests-btn-reject";
      rejectBtn.textContent = "Reject";
      rejectBtn.addEventListener("click", () =>
        mutateRequest(req.id, "reject")
      );

      actionsWrap.appendChild(editBtn);
      actionsWrap.appendChild(approveBtn);
      actionsWrap.appendChild(rejectBtn);

      actionsTd.appendChild(actionsWrap);
      tr.appendChild(actionsTd);

      tableBody.appendChild(tr);
    });
  }

  async function loadRequests() {
    errorNotice.classList.add("hidden");

    const urls = ["/admin/requests/data"];

    for (const url of urls) {
      try {
        const res = await fetch(url, {
          credentials: "include",
          headers: { Accept: "application/json" },
        });
        if (!res.ok) continue;

        const data = await res.json();
        const raw =
          Array.isArray(data) ? data : data.requests || data.items || [];

        const normalized = raw.map((req) => {
          const tags =
            Array.isArray(req.tags)
              ? req.tags
              : typeof req.tags === "string"
              ? req.tags
                  .split(",")
                  .map((t) => t.trim())
                  .filter(Boolean)
              : [];

          let submitted_at_human = req.submitted_at_human;
          const created =
            req.created_at || req.createdAt || req.submitted_at || null;

          if (!submitted_at_human && created) {
            const d = new Date(created);
            if (!Number.isNaN(d.getTime())) {
              submitted_at_human = d.toLocaleDateString(undefined, {
                year: "numeric",
                month: "short",
                day: "2-digit",
              });
            }
          }

          return {
            ...req,
            tags,
            submitted_at_human,
          };
        });

        renderTable(normalized);
        return;
      } catch (_err) {
      }
    }

    errorNotice.classList.remove("hidden");
  }

  function mutateRequest(id, action) {
    const path =
      action === "approve"
        ? `/admin/requests/${id}/approve`
        : `/admin/requests/${id}/reject`;

    fetch(path, {
      method: "POST",
      credentials: "include",
    })
      .then((res) => {
        if (!res.ok) throw new Error("bad status");
        return res.json().catch(() => null);
      })
      .then(() => {
        loadRequests();
      })
      .catch(() => {
        errorNotice.classList.remove("hidden");
      });
  }

  editForm.addEventListener("submit", (e) => {
    e.preventDefault();
    if (!adminState.editingId) return;

    editError.classList.add("hidden");

    const tagsRaw = editForm.elements["tags"].value || "";
    const tags = tagsRaw
      .split(",")
      .map((t) => t.trim())
      .filter(Boolean);

    const payload = {
      server_name: editForm.elements["server_name"].value.trim(),
      url: editForm.elements["url"].value.trim() || null,
      logo_url: editForm.elements["logo_url"].value.trim() || null,
      description: editForm.elements["description"].value.trim(),
      tags,
      owner_name: editForm.elements["owner_name"].value.trim(),
      owner_discord: editForm.elements["owner_discord"].value.trim(),
    };

    fetch(`/admin/requests/${adminState.editingId}/update`, {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
    })
      .then((res) => {
        if (!res.ok) throw new Error("bad status");
        return res.json().catch(() => null);
      })
      .then(() => {
        closeEditModal();
        loadRequests();
      })
      .catch(() => {
        editError.classList.remove("hidden");
      });
  });

  closeBtn.addEventListener("click", closeEditModal);
  cancelBtn.addEventListener("click", closeEditModal);

  overlay.addEventListener("click", (e) => {
    if (e.target === overlay) {
      closeEditModal();
    }
  });

  loadRequests();
}
