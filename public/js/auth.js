export function initAuth() {
  const loginBtn = document.getElementById("discord-login");
  const userContainer = document.getElementById("nav-user");
  const avatarEl = document.getElementById("nav-user-avatar");
  const triggerEl = document.getElementById("nav-user-trigger");
  const menuEl = document.getElementById("nav-user-menu");
  const logoutBtn = document.getElementById("nav-logout");
  const adminBtn = document.getElementById("nav-admin");

  if (userContainer) {
    userContainer.classList.add("hidden");
  }
  if (adminBtn) {
    adminBtn.classList.add("hidden");
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
        if (adminBtn) adminBtn.classList.add("hidden");
        return;
      }

      if (loginBtn) loginBtn.classList.add("hidden");
      userContainer.classList.remove("hidden");

      if (avatarEl && data.avatar_url) {
        avatarEl.src = data.avatar_url;
        avatarEl.alt = data.username || "Discord avatar";
      }

      if (adminBtn) {
        if (data.is_admin) {
          adminBtn.classList.remove("hidden");
          adminBtn.onclick = (e) => {
            e.preventDefault();
            window.location.href = "/admin/requests";
          };
        } else {
          adminBtn.classList.add("hidden");
        }
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
      if (adminBtn) adminBtn.classList.add("hidden");
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
