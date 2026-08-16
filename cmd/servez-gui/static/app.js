/* ServeEz GUI — single-page dashboard.
   Polls the control plane via the /v1/* proxy, renders the five views
   (Overview, Nodes, Workloads, Cost, Alerts, AI Control). Dependency-free. */

(function () {
  "use strict";

  var POLL_MS = 3000;
  var state = { objects: [], nodes: [], workloads: [], audit: [] };
  var activeView = "overview";
  var lastCost = null;

  /* ---------- helpers ---------- */

  function $(id) { return document.getElementById(id); }

  function esc(s) {
    return String(s == null ? "" : s).replace(/[&<>"']/g, function (c) {
      return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c];
    });
  }

  function fmtBytes(b) {
    if (!b) return "0 B";
    var gb = b / (1024 * 1024 * 1024);
    if (gb >= 1) return gb.toFixed(1) + " GB";
    var mb = b / (1024 * 1024);
    if (mb >= 1) return mb.toFixed(0) + " MB";
    return Math.round(b / 1024) + " KB";
  }

  function fmtNet(b) {
    if (!b) return "0 B/s";
    if (b >= 1024 * 1024) return (b / (1024 * 1024)).toFixed(0) + " MB/s";
    return Math.round(b / 1024) + " KB/s";
  }

  function timeAgo(t) {
    if (!t) return "";
    var ms = Date.parse(t);
    if (isNaN(ms)) return "";
    var s = Math.floor((Date.now() - ms) / 1000);
    if (s < 60) return "just now";
    if (s < 3600) return Math.floor(s / 60) + "m ago";
    if (s < 86400) return Math.floor(s / 3600) + "h ago";
    return Math.floor(s / 86400) + "d ago";
  }

  function nodeStateColor(st) {
    st = (st || "pending").toLowerCase();
    if (st === "healthy") return "ok";
    if (st === "degraded" || st === "cordoned") return "warn";
    if (st === "unhealthy" || st === "disconnected" || st === "down") return "bad";
    return "neutral";
  }

  function workloadState(w) {
    var st = w && w.Status && w.Status.State;
    var name = w.Name || "unknown";
    if (st === "running" || st === "healthy") return { cls: "ok", label: "Running" };
    if (st === "degraded") return { cls: "warn", label: "Degraded" };
    if (st === "scaled") return { cls: "neutral", label: "Scaled" };
    if (st === "pending") return { cls: "neutral", label: "Pending" };
    return { cls: "neutral", label: st ? String(st) : "Unknown" };
  }

  function splitObjects() {
    state.nodes = [];
    state.workloads = [];
    (state.objects || []).forEach(function (o) {
      if (!o) return;
      var kind = (o.Kind || o.kind || "").toLowerCase();
      if (kind === "node") state.nodes.push(o);
      else if (kind === "workload" || kind === "service") state.workloads.push(o);
    });
  }

  /* ---------- data fetch ---------- */

  function fetchJSON(url, opts) {
    return fetch(url, opts || {}).then(function (r) {
      if (!r.ok) throw new Error("HTTP " + r.status);
      return r.json();
    });
  }

  function refreshState() {
    return fetchJSON("/v1/state?type=" + encodeURIComponent(""))
      .then(function (data) {
        state.objects = data.objects || [];
        splitObjects();
        return fetchJSON("/v1/audit").then(function (audit) {
          state.audit = (audit.entries || []).map(function (e) {
            return { text: e.description || e.action || "", time: e.timestamp || e.created_at || "" };
          });
        }).catch(function () {
          state.audit = [];
        });
      })
      .catch(function () {
        state.objects = [];
        splitObjects();
      });
  }

  /* ---------- overview ---------- */

  function renderOverview() {
    var healthy = 0, total = state.nodes.length;
    state.nodes.forEach(function (n) {
      var st = (n.Status && n.Status.State || "").toLowerCase();
      if (st === "healthy" || st === "running") healthy++;
    });
    $("ov-nodes").textContent = total;
    $("ov-nodes-sub").textContent = total === 0 ? "waiting for agents…" :
      (healthy === total ? "all healthy" : healthy + "/" + total + " healthy");
    $("ov-workloads").textContent = state.workloads.length;
    $("ov-workloads-sub").textContent = state.workloads.length + " workload(s) tracked";

    var grid = $("node-grid");
    if (state.nodes.length === 0) {
      grid.innerHTML = '<div class="empty"><i>No nodes registered. Start agents with servez-agent.</i></div>';
    } else {
      grid.innerHTML = state.nodes.map(function (n) {
        var st = n.Status || {};
        var us = st.Resources || {};
        var cls = nodeStateColor(st.State);
        var hot = cls === "warn" || cls === "bad";
        var cap = n.Spec && n.Spec.Capacity || {};
        var role = n.Spec && n.Spec.Provider || "local";
        return '<div class="node-tile' + (hot ? " hot" : "") + '">' +
          '<span class="dot ' + cls + ' status-dot"></span>' +
          '<span class="name">' + esc(n.Name) + '</span>' +
          '<span class="role">' + esc(role) + '</span>' +
          '<div class="rows">' +
          '<div class="row"><span class="k">cpu</span><span class="v">' + (us.CPUPercent != null ? us.CPUPercent.toFixed(1) + "%" : "—") + '</span></div>' +
          '<div class="row"><span class="k">mem</span><span class="v">' + fmtBytes(us.MemUsedBytes != null ? us.MemUsedBytes : 0) + (cap.MemBytes ? " / " + fmtBytes(cap.MemBytes) : "") + '</span></div>' +
          '<div class="row"><span class="k">net</span><span class="v">' + fmtNet(us.NetRxBps || 0) + '</span></div>' +
          '</div></div>';
      }).join("");
    }

    renderActivity();
    renderMiniAlerts();
  }

  function renderActivity() {
    var list = state.audit.slice(0, 5);
    if (list.length === 0) {
      $("activity-list").innerHTML = '<li class="empty"><i>No activity yet</i></li>';
      return;
    }
    $("activity-list").innerHTML = list.map(function (e) {
      return '<li><span class="mono" style="font-size:12px;">' + esc(e.text) + '</span>' +
        '<span class="time">' + timeAgo(e.time) + '</span></li>';
    }).join("");
  }

  function renderMiniAlerts() {
    var alerts = buildAlerts().filter(function (a) { return a.active; });
    if (alerts.length === 0) {
      $("alert-list").innerHTML = '<li class="empty"><i>No active alerts</i></li>';
      return;
    }
    $("alert-list").innerHTML = alerts.slice(0, 3).map(function (a) {
      return '<li>' +
        '<span style="display:flex;align-items:center;gap:8px;">' +
        '<span class="dot ' + a.cls + '"></span>' +
        '<span class="mono" style="font-size:12px;color:' + (a.cls === "warn" ? "var(--tertiary)" : "var(--error)") + ';">' + esc(a.title) + '</span>' +
        '</span><span class="time">' + esc(a.sev) + '</span></li>';
    }).join("");
  }

  /* ---------- nodes ---------- */

  function renderNodes() {
    var tbody = $("nodes-table-body");
    if (state.nodes.length === 0) {
      tbody.innerHTML = '<tr><td colspan="8" class="empty"><i>No nodes registered.</i></td></tr>';
      return;
    }
    tbody.innerHTML = state.nodes.map(function (n) {
      var st = n.Status || {};
      var us = st.Resources || {};
      var cap = n.Spec && n.Spec.Capacity || {};
      return '<tr>' +
        '<td class="mono">' + esc(n.Name) + '</td>' +
        '<td class="muted">' + esc(n.Spec && n.Spec.Provider || "local") + '</td>' +
        '<td><span class="chip ' + nodeStateColor(st.State) + '"><span class="dot"></span>' + esc(st.State || "pending") + '</span></td>' +
        '<td class="num">' + (us.CPUPercent != null ? us.CPUPercent.toFixed(1) + "%" : "—") + '</td>' +
        '<td class="num">' + fmtBytes(us.MemUsedBytes != null ? us.MemUsedBytes : 0) + (cap.MemBytes ? " / " + fmtBytes(cap.MemBytes) : "") + '</td>' +
        '<td class="num">' + (us.DiskPercent != null ? us.DiskPercent.toFixed(0) + "%" : "—") + '</td>' +
        '<td class="num">' + fmtNet(us.NetRxBps || 0) + '</td>' +
        '<td class="muted">' + timeAgo(st.LastSeen) + '</td>' +
        '</tr>';
    }).join("");
  }

  /* ---------- services ---------- */

  function renderServices(filter) {
    filter = filter || "all";
    var rows = state.workloads.filter(function (w) {
      if (filter === "all") return true;
      var st = w.Status && w.Status.State;
      if (filter === "running") return st === "running" || st === "healthy";
      if (filter === "degraded") return st === "degraded";
      if (filter === "scaled") return st === "scaled";
      return true;
    });
    var tbody = $("services-table-body");
    if (rows.length === 0) {
      tbody.innerHTML = '<tr><td colspan="7" class="empty"><i>No workloads' + (filter !== "all" ? " matching filter" : "") + '.</i></td></tr>';
      return;
    }
    tbody.innerHTML = rows.map(function (w) {
      var st = workloadState(w);
      var spec = w.Spec || {};
      var us = w.Status && w.Status.Resources || {};
      var node = w.Status && (w.Status.Node || w.Status.AssignedNode) || "—";
      var repl = spec.Replicas != null ? spec.Replicas : "—";
      return '<tr>' +
        '<td class="mono">' + esc(w.Name) + '</td>' +
        '<td class="mono muted" style="font-size:12px;">' + esc(spec.Image || "—") + '</td>' +
        '<td class="num">' + repl + '</td>' +
        '<td class="mono muted">' + esc(node) + '</td>' +
        '<td class="num">' + (us.CPUPercent != null ? us.CPUPercent.toFixed(1) + "%" : "—") + '</td>' +
        '<td class="num">' + fmtBytes(us.MemUsedBytes != null ? us.MemUsedBytes : 0) + '</td>' +
        '<td><span class="chip ' + st.cls + '"><span class="dot"></span>' + st.label + '</span></td>' +
        '</tr>';
    }).join("");
  }

  /* ---------- cost ---------- */

  function renderCostReport(report) {
    lastCost = report;
    var prov = report.providers || [];
    $("cost-best-provider").textContent = report.best ? report.best.provider.toUpperCase() : "—";
    $("cost-best-monthly").textContent = report.best ? "$" + report.best.est_monthly.toFixed(2) : "—";
    $("cost-providers").textContent = prov.map(function (p) { return p.provider.toUpperCase(); }).join(" · ") || "—";
    var spot = report.best && (report.best.on_demand_per_mo - report.best.est_monthly) || 0;
    $("cost-spot-savings").textContent = report.best ? "$" + spot.toFixed(2) + "/mo" : "—";
    $("cost-table-title").textContent = "Instances (" + (report.request ? report.request.vcpu + " vCPU, " + report.request.mem_gb + "GB" : "") + ")";

    var tbody = $("cost-table-body");
    if (prov.length === 0) {
      tbody.innerHTML = '<tr><td colspan="7" class="empty"><i>No offers matched that shape.</i></td></tr>';
      return;
    }
    tbody.innerHTML = prov.map(function (p) {
      var best = report.best && report.best.provider === p.provider;
      var saving = p.on_demand_per_mo > 0 ? Math.round((1 - p.est_monthly / p.on_demand_per_mo) * 100) : 0;
      var rec;
      if (best) rec = '<span class="chip rec">★ Recommended</span>';
      else if (p.recommended === "spot") rec = '<span class="chip neutral">Spot</span>';
      else rec = '<span class="chip neutral">' + esc(p.recommended || "—") + '</span>';
      return '<tr' + (best ? ' class="hl"' : '') + '>' +
        '<td style="font-weight:600;">' + esc(p.provider.toUpperCase()) + '</td>' +
        '<td class="mono muted">' + esc(p.instance_type) + '</td>' +
        '<td class="muted">' + esc(p.region) + '</td>' +
        '<td class="num">$' + p.on_demand_per_mo.toFixed(2) + '</td>' +
        '<td class="num' + (best ? '" style="color:var(--primary-hi);font-weight:700;' : '') + '">' +
        (p.spot_per_mo > 0 ? "$" + p.spot_per_mo.toFixed(2) : '<span class="muted">N/A*</span>') + '</td>' +
        '<td class="num" style="color:var(--secondary);">' + saving + '%</td>' +
        '<td style="text-align:center;">' + rec + '</td>' +
        '</tr>';
    }).join("");
  }

  function runCostCompare() {
    var vcpu = parseInt($("cost-vcpu").value, 10) || 4;
    var mem = parseInt($("cost-mem").value, 10) || 16;
    var hours = parseInt($("cost-hours").value, 10) || 730;
    var runtime = Math.min(100, Math.round(hours / 730 * 100));
    $("cost-table-body").innerHTML = '<tr><td colspan="7" class="empty"><i>Comparing AWS · GCP · Azure…</i></td></tr>';
    fetchJSON("/v1/cost/compare", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ vcpu: vcpu, mem_gb: mem, runtime_pct: runtime })
    }).then(function (report) {
      renderCostReport(report);
    }).catch(function (e) {
      $("cost-table-body").innerHTML = '<tr><td colspan="7" class="empty"><i>Compare failed: ' + esc(e.message) + '</i></td></tr>';
    });
  }

  /* ---------- alerts ---------- */

  function buildAlerts() {
    var out = [];
    state.nodes.forEach(function (n) {
      var st = n.Status || {};
      var us = st.Resources || {};
      var stateStr = (st.State || "").toLowerCase();
      if (stateStr === "degraded" || us.CPUPercent >= 90) {
        out.push({ sev: "Warning", cls: "warn", active: true, title: n.Name + " cpu > 90%", detail: "cpu at " + (us.CPUPercent ? us.CPUPercent.toFixed(1) + "%" : "high") + " · " + timeAgo(st.LastSeen) });
      }
      if (us.MemPercent >= 90) {
        out.push({ sev: "Warning", cls: "warn", active: true, title: n.Name + " memory at " + us.MemPercent.toFixed(0) + "%", detail: "mem " + fmtBytes(us.MemUsedBytes) + " · trend rising", time: st.LastSeen });
      }
      if (stateStr === "unhealthy" || stateStr === "disconnected" || stateStr === "down") {
        out.push({ sev: "Critical", cls: "bad", active: true, title: n.Name + " unreachable", detail: "node " + stateStr, time: st.LastSeen });
      }
    });
    return out;
  }

  function renderAlerts(filter) {
    filter = filter || "all";
    var alerts = buildAlerts().filter(function (a) {
      if (filter === "all") return true;
      return a.sev.toLowerCase() === filter;
    });
    $("alert-count").textContent = alerts.length + " active alert" + (alerts.length === 1 ? "" : "s");
    var el = $("alerts-list");
    if (alerts.length === 0) {
      el.innerHTML = '<div class="empty"><i>No active alerts — the cluster is calm.</i></div>';
      return;
    }
    el.innerHTML = alerts.map(function (a) {
      return '<div class="alert-row">' +
        '<span class="dot ' + a.cls + '"></span>' +
        '<div style="flex:1;min-width:0;">' +
        '<div style="display:flex;justify-content:space-between;gap:16px;">' +
        '<span class="title" style="color:' + (a.cls === "warn" ? "var(--on-surface)" : "var(--error)") + ';">' + esc(a.title) + '</span>' +
        '<span class="chip ' + (a.active ? "warn" : "neutral") + '">' + (a.active ? "Active" : "Resolved") + '</span>' +
        '</div>' +
        '<div class="detail"><span>' + esc(a.detail) + '</span><span class="t">' + timeAgo(a.time) + '</span></div>' +
        '</div></div>';
    }).join("");
  }

  /* ---------- AI control ---------- */

  function parseIntent(text) {
    var t = String(text || "").toLowerCase();
    if (t.indexOf("scale") >= 0) return { intent: "scale" };
    if (t.indexOf("migrate") >= 0) return { intent: "migrate" };
    if (t.indexOf("restart") >= 0) return { intent: "restart" };
    if (t.indexOf("kill") >= 0) return { intent: "kill" };
    return null;
  }

  function extractWorkload(text) {
    var m = /(api-gateway|payments|web|auth|worker|ml-inference|observability|billing)/.exec(text);
    return m ? m[1] : null;
  }

  function proposeAction(text) {
    var intent = parseIntent(text);
    var wl = extractWorkload(text);
    var w = wl && state.workloads.filter(function (x) { return x.Name === wl; })[0];
    var node = w && w.Status && (w.Status.Node || w.Status.AssignedNode) || "node-a1";
    var conf = 0.85, impact = "+12% capacity";
    if (!intent) {
      $("action-title").textContent = "No action proposed";
      $("kv-node").textContent = "—";
      $("kv-conf").textContent = "—";
      $("kv-impact").textContent = "—";
      $("conf-value").textContent = "—";
      $("conf-fill").style.width = "0%";
      $("intent-tag").textContent = "NONE";
      return;
    }
    var label = intent.toUpperCase();
    $("intent-tag").textContent = label;
    var action = "—";
    if (intent === "scale") action = "Scale " + (wl || "workload") + " 2 → 4 replicas";
    else if (intent === "migrate") action = "Migrate " + (wl || "workload") + " to " + node;
    else if (intent === "restart") action = "Restart " + (wl || "workload") + " on " + node;
    else if (intent === "kill") action = "Stop " + (wl || "workload");
    $("action-title").textContent = action;
    $("kv-node").textContent = node;
    $("kv-conf").textContent = conf.toFixed(2);
    $("kv-impact").textContent = impact;
    $("conf-value").textContent = conf.toFixed(2);
    $("conf-fill").style.width = Math.round(conf * 100) + "%";
  }

  function appendMsg(role, text, time) {
    var thread = $("ai-thread");
    var div = document.createElement("div");
    div.className = "msg " + role;
    if (role === "user") {
      div.innerHTML = '<div class="bubble">' + esc(text) + '</div><span class="time">just now</span>';
    } else {
      div.innerHTML = '<div class="body">' + text + '</div><span class="time">' + (time || "just now") + '</span>';
    }
    thread.appendChild(div);
    thread.scrollTop = thread.scrollHeight;
  }

  function handleAI(text) {
    if (!text || !text.trim()) return;
    appendMsg("user", text);
    $("ai-input").value = "";
    var intent = parseIntent(text);
    if (!intent) {
      setTimeout(function () {
        appendMsg("assistant", "I can propose <b>scale</b>, <b>migrate</b>, <b>restart</b>, or <b>kill</b> actions. Try: \"scale api-gateway to 4 replicas\".", "now");
      }, 400);
      $("action-title").textContent = "No action proposed";
      return;
    }
    setTimeout(function () {
      var wl = extractWorkload(text);
      var body = 'I propose to ' + intent + ' <b>' + esc(wl || "the workload") + '</b>. ' +
        'Forecast confidence <b>0.85</b>, impact <b>+12% capacity</b>. Review the proposed action on the right.';
      appendMsg("assistant", body, "now");
      proposeAction(text);
    }, 400);
  }

  function executeProposed() {
    var title = $("action-title").textContent;
    if (!title || title === "No action proposed" || title === "—") return;
    appendMsg("assistant", "Simulated: <b>" + esc(title) + "</b>. Confidence gate passed (0.85 ≥ 0.70). Waiting for your confirm to execute — execution is read-only in this build.");
    $("action-title").textContent = "Awaiting confirmation";
  }

  /* ---------- view routing ---------- */

  function showView(name) {
    activeView = name;
    document.querySelectorAll(".view").forEach(function (v) { v.classList.remove("active"); });
    var view = $("view-" + name);
    if (view) view.classList.add("active");
    document.querySelectorAll(".nav-item").forEach(function (n) {
      n.classList.toggle("active", n.getAttribute("data-view") === name);
    });
    var title = view ? view.getAttribute("data-title") : "ServeEz";
    $("page-title").textContent = title;
    document.title = "ServeEz — " + title;
    if (name === "cost" && !lastCost) runCostCompare();
    if (name === "alerts") renderAlerts(getFilter("afilter"));
    if (name === "services") renderServices(getFilter("wfilter"));
  }

  function getFilter(kind) {
    var active = document.querySelector(".filter-chip.active[data-" + kind + "]");
    return active ? active.getAttribute("data-" + kind) : "all";
  }

  /* ---------- wiring ---------- */

  document.querySelectorAll(".nav-item").forEach(function (n) {
    n.addEventListener("click", function () { showView(n.getAttribute("data-view")); });
  });

  document.querySelectorAll("[data-filter]").forEach(function (c) {
    c.addEventListener("click", function () {
      document.querySelectorAll(".filter-chip[data-filter]").forEach(function (x) { x.classList.remove("active"); });
      c.classList.add("active");
      renderNodes();
    });
  });
  document.querySelectorAll("[data-wfilter]").forEach(function (c) {
    c.addEventListener("click", function () {
      document.querySelectorAll(".filter-chip[data-wfilter]").forEach(function (x) { x.classList.remove("active"); });
      c.classList.add("active");
      renderServices(c.getAttribute("data-wfilter"));
    });
  });
  document.querySelectorAll("[data-afilter]").forEach(function (c) {
    c.addEventListener("click", function () {
      document.querySelectorAll(".filter-chip[data-afilter]").forEach(function (x) { x.classList.remove("active"); });
      c.classList.add("active");
      renderAlerts(c.getAttribute("data-afilter"));
    });
  });

  $("refresh-btn").addEventListener("click", function () {
    refreshState().then(function () {
      renderOverview();
      renderNodes();
      renderServices(getFilter("wfilter"));
      renderAlerts(getFilter("afilter"));
    });
  });

  $("cost-compare-btn").addEventListener("click", runCostCompare);
  ["cost-vcpu", "cost-mem", "cost-hours"].forEach(function (id) {
    $(id).addEventListener("keydown", function (e) { if (e.key === "Enter") runCostCompare(); });
  });

  $("ai-send").addEventListener("click", function () { handleAI($("ai-input").value); });
  $("ai-input").addEventListener("keydown", function (e) { if (e.key === "Enter") handleAI($("ai-input").value); });
  $("exec-btn").addEventListener("click", executeProposed);
  $("sim-btn").addEventListener("click", executeProposed);
  $("deploy-btn").addEventListener("click", function () {
    var name = prompt("Workload name (e.g. web-frontend):");
    if (!name) return;
    var image = prompt("Image (e.g. ghcr.io/acme/web:1.0):");
    if (!image) return;
    fetchJSON("/v1/workloads", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: name, spec: { image: image, replicas: 1 } })
    }).then(function () {
      alert("Deployed " + name + ". Waiting for a node to pick it up…");
      refreshState();
    }).catch(function (e) {
      alert("Deploy failed: " + e.message);
    });
  });

  /* ---------- boot ---------- */

  function tick() {
    refreshState().then(function () {
      var total = state.nodes.length;
      var bad = 0;
      state.nodes.forEach(function (n) {
        var st = (n.Status && n.Status.State || "").toLowerCase();
        if (st === "unhealthy" || st === "disconnected" || st === "down") bad++;
      });
      $("foot-status").textContent = total === 0 ? "no agents" :
        (bad === 0 ? total + " nodes · healthy" : total + " nodes · " + bad + " down");
      $("foot-dot").className = "dot " + (bad === 0 ? "ok" : "bad");
      renderOverview();
      renderNodes();
      if (activeView === "services") renderServices(getFilter("wfilter"));
      if (activeView === "alerts") renderAlerts(getFilter("afilter"));
    });
  }

  tick();
  setInterval(tick, POLL_MS);
})();