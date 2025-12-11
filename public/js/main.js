// public/js/main.js

import { initTheme } from "./theme.js";
import { initAuth } from "./auth.js";
import { initLeaderboard, handleVoteClick } from "./leaderboard.js";
import { initServerDetail } from "./server-detail.js";
import { initListPage } from "./list.js";
import { initAdminRequests } from "./admin-requests.js";

document.addEventListener("DOMContentLoaded", () => {
  initTheme();
  initAuth();

  const adminRoot = document.getElementById("admin-requests-root");
  const detailRoot = document.getElementById("server-detail");
  const listForm = document.getElementById("server-request-form");
  const serverGridEl = document.getElementById("server-grid");

  if (adminRoot) {
    initAdminRequests();
  } else if (detailRoot) {
    initServerDetail();
  } else if (listForm) {
    initListPage();
  } else if (serverGridEl) {
    initLeaderboard();
  }
});

document.addEventListener("click", (event) => {
  const target = event.target;
  if (!(target instanceof HTMLElement)) return;

  if (target.matches(".vote-button")) {
    event.preventDefault();
    handleVoteClick(target);
  }
});
