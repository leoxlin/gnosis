import Alpine from "alpinejs";
import "./styles.css";

// aborted marks requests superseded by a newer navigation so views ignore them.
const aborted = () => {
  const error = new Error("request superseded");
  error.aborted = true;
  return error;
};

// api fetches one JSON endpoint with a timeout and surfaces short, actionable
// errors: non-JSON error bodies report the HTTP status, network failures read
// as an unreachable server, and HTTP errors carry a status for 404/410 checks.
// options: timeout (ms, longer default for the slower vector search), signal
// (caller abort), method, body (JSON-encoded when present).
const api = async (path, options = {}) => {
  const timeout = options.timeout || (path.startsWith("/api/v1/search") ? 30000 : 10000);
  const controller = new AbortController();
  if (options.signal) {
    if (options.signal.aborted) controller.abort();
    else options.signal.addEventListener("abort", () => controller.abort(), { once: true });
  }
  const timer = setTimeout(() => controller.abort("timeout"), timeout);
  let response;
  try {
    response = await fetch(path, {
      method: options.method || "GET",
      headers: options.body === undefined ? undefined : { "Content-Type": "application/json" },
      body: options.body === undefined ? undefined : JSON.stringify(options.body),
      signal: controller.signal,
    });
  } catch (error) {
    clearTimeout(timer);
    if (controller.signal.aborted) {
      if (controller.signal.reason === "timeout") {
        throw new Error(`Request timed out after ${Math.round(timeout / 1000)}s.`);
      }
      throw aborted();
    }
    throw new Error("Server unreachable. Check that gnosis serve is running.");
  }
  let data = null;
  try {
    data = await response.json();
  } catch (error) {
    clearTimeout(timer);
    if (controller.signal.aborted) throw aborted();
    if (!response.ok) {
      throw new Error(`Request failed (HTTP ${response.status}) with a non-JSON response.`);
    }
    throw new Error(`The server returned malformed JSON (HTTP ${response.status}).`);
  }
  clearTimeout(timer);
  if (!response.ok) {
    const failure = new Error((data && data.error) || `Request failed (HTTP ${response.status}).`);
    failure.status = response.status;
    throw failure;
  }
  return data;
};

const views = ["dashboard", "graph", "pages", "concepts", "search", "vaults"];

const parseHash = () => {
  const hash = location.hash.replace(/^#\/?/, "");
  const [name = "", ...rest] = hash.split("/");
  if (name === "page" && rest.length > 0) {
    return { name: "page", uri: decodeURIComponent(rest.join("/")) };
  }
  return { name: views.includes(name) ? name : "dashboard" };
};

// lazy runs load the first time the named view becomes active.
const lazy = (component, view, load) => {
  component.$watch("$store.app.route.name", (name) => {
    if (name === view) load();
  });
  if (Alpine.store("app").route.name === view) load();
};

const shortRevision = (revision) => (revision || "").replace(/^sha256:/, "").slice(0, 12);

document.addEventListener("alpine:init", () => {
  Alpine.store("app", {
    route: parseHash(),
    pendingQuestion: "",
    is(name) {
      return this.route.name === name;
    },
    openPage(uri) {
      location.hash = "#/page/" + encodeURIComponent(uri);
    },
    openSearch(question) {
      this.pendingQuestion = question;
      location.hash = "#/search";
    },
  });

  Alpine.data("dashboardView", () => ({
    question: "",
    pulse: null,
    pulseError: "",
    pulseLoading: false,
    health: null,
    healthError: "",
    healthLoading: false,
    changes: [],
    changesCursor: "",
    changesError: "",
    changesLoading: false,
    changesLoaded: false,
    changesNotice: "",

    init() {
      lazy(this, "dashboard", () => this.load());
    },

    short: shortRevision,

    // load fires every panel that has not loaded yet; panels run in parallel
    // and fail independently so one slow endpoint never blocks the others.
    load() {
      if (!this.pulse && !this.pulseLoading) this.loadPulse();
      if (!this.health && !this.healthLoading) this.loadHealth();
      if (!this.changesLoaded && !this.changesLoading) this.loadChanges();
    },

    async loadPulse() {
      this.pulseLoading = true;
      this.pulseError = "";
      try {
        const [pages, vaults, concepts] = await Promise.all([
          api("/api/v1/pages"),
          api("/api/v1/vaults"),
          api("/api/v1/concepts"),
        ]);
        const list = (pages.pages || []).map((page) => ({ type: "", ...page }));
        const byType = {};
        for (const page of list) {
          const type = page.type || "unknown";
          byType[type] = (byType[type] || 0) + 1;
        }
        this.pulse = {
          pages: list.length,
          vaults: (vaults.vaults || []).length,
          types: (concepts.concept_types || []).length,
          byType: Object.entries(byType).sort((a, b) => b[1] - a[1]),
        };
      } catch (error) {
        this.pulseError = error.message;
      } finally {
        this.pulseLoading = false;
      }
    },

    async loadHealth() {
      this.healthLoading = true;
      this.healthError = "";
      try {
        const result = await api("/api/v1/audit/knowledge", { method: "POST", body: {} });
        result.findings = (result.findings || []).map((finding) => ({ uris: [], ...finding }));
        this.health = result;
      } catch (error) {
        this.healthError = error.message;
      } finally {
        this.healthLoading = false;
      }
    },

    async loadChanges(cursor = "") {
      this.changesLoading = true;
      this.changesError = "";
      try {
        const params = new URLSearchParams({ limit: "20" });
        if (cursor) params.set("cursor", cursor);
        const result = await api("/api/v1/changes?" + params);
        const batch = (result.changes || []).map((change) => ({
          classification: "changed",
          uri: "",
          before_revision: "",
          after_revision: "",
          ...change,
        }));
        this.changes = cursor ? this.changes.concat(batch) : batch;
        this.changesCursor = result.next_cursor || "";
        this.changesLoaded = true;
      } catch (error) {
        if (cursor && error.status === 410) {
          this.changesCursor = "";
          this.changesLoading = false;
          this.changesNotice = "The changes cursor expired; reloaded the latest changes.";
          return this.loadChanges();
        }
        this.changesError = error.message;
      } finally {
        this.changesLoading = false;
      }
    },

    retryChanges() {
      this.changesNotice = "";
      this.loadChanges();
    },

    goSearch() {
      const question = this.question.trim();
      if (question) this.$store.app.openSearch(question);
    },
  }));

  Alpine.data("graphView", () => ({
    nodes: [],
    edges: [],
    types: [],
    query: "",
    type: "",
    loaded: false,
    error: "",
    frames: 0,
    context: null,

    init() {
      this.canvas = this.$refs.canvas;
      window.addEventListener("resize", () => this.resize());
      lazy(this, "graph", () => {
        this.$nextTick(() => {
          this.resize();
          if (!this.loaded) this.load();
        });
      });
    },

    color(value) {
      let hash = 0;
      for (const character of value) hash = (hash * 31 + character.charCodeAt(0)) | 0;
      return `hsl(${Math.abs(hash) % 360} 48% 64%)`;
    },

    matches(node) {
      const query = this.query.toLowerCase();
      const text = `${node.title} ${node.type} ${node.description || ""}`.toLowerCase();
      return (!this.type || node.type === this.type) && (!query || text.includes(query));
    },

    visible() {
      return this.nodes.filter((node) => this.matches(node));
    },

    async load() {
      this.error = "";
      try {
        const graph = await api("/api/v1/graph");
        const nodes = (graph.nodes || []).map((node) => {
          const normalized = { type: "", description: "", uri: "", ...node };
          normalized.title = normalized.title || normalized.uri;
          return normalized;
        });
        const count = Math.max(1, nodes.length);
        this.nodes = nodes.map((node, index) => {
          const angle = index * 2.399963229728653;
          const radius = 0.08 + 0.34 * Math.sqrt((index + 1) / count);
          return { ...node, x: 0.5 + Math.cos(angle) * radius, y: 0.5 + Math.sin(angle) * radius };
        });
        const byURI = new Map(this.nodes.map((node) => [node.uri, node]));
        this.edges = (graph.edges || [])
          .filter((edge) => edge.from && edge.to && edge.from.uri && edge.to.uri)
          .map((edge) => ({
            ...edge,
            fromNode: byURI.get(edge.from.uri),
            toNode: byURI.get(edge.to.uri),
          }));
        this.types = [...new Set(this.nodes.map((node) => node.type))].sort();
        this.loaded = true;
        this.settle();
      } catch (error) {
        this.error = error.message;
      }
    },

    resize() {
      if (!this.canvas || !this.canvas.clientWidth) return;
      const box = this.canvas.getBoundingClientRect();
      const ratio = window.devicePixelRatio || 1;
      this.canvas.width = Math.max(1, Math.floor(box.width * ratio));
      this.canvas.height = Math.max(1, Math.floor(box.height * ratio));
      this.context = this.canvas.getContext("2d");
      this.context.setTransform(ratio, 0, 0, ratio, 0, 0);
      this.draw();
    },

    settle() {
      if (!this.nodes.length || this.frames > 180) {
        this.draw();
        return;
      }
      for (let i = 0; i < this.nodes.length; i++) {
        this.nodes[i].dx = 0;
        this.nodes[i].dy = 0;
        for (let j = i + 1; j < this.nodes.length; j++) {
          const x = this.nodes[j].x - this.nodes[i].x;
          const y = this.nodes[j].y - this.nodes[i].y;
          const distance = Math.max(0.002, x * x + y * y);
          const force = Math.min(0.00012 / distance, 0.012);
          this.nodes[i].dx -= x * force;
          this.nodes[i].dy -= y * force;
          this.nodes[j].dx += x * force;
          this.nodes[j].dy += y * force;
        }
      }
      for (const edge of this.edges) {
        if (!edge.fromNode || !edge.toNode) continue;
        const x = edge.toNode.x - edge.fromNode.x;
        const y = edge.toNode.y - edge.fromNode.y;
        const force = Math.max(0, Math.hypot(x, y) - 0.16) * 0.018;
        edge.fromNode.dx += x * force;
        edge.fromNode.dy += y * force;
        edge.toNode.dx -= x * force;
        edge.toNode.dy -= y * force;
      }
      for (const node of this.nodes) {
        node.dx += (0.5 - node.x) * 0.00035;
        node.dy += (0.5 - node.y) * 0.00035;
        node.x = Math.min(0.94, Math.max(0.06, node.x + node.dx));
        node.y = Math.min(0.91, Math.max(0.09, node.y + node.dy));
      }
      this.frames++;
      this.draw();
      requestAnimationFrame(() => this.settle());
    },

    draw() {
      if (!this.context) return;
      const box = this.canvas.getBoundingClientRect();
      const context = this.context;
      context.clearRect(0, 0, box.width, box.height);
      const visible = new Set(this.visible().map((node) => node.uri));
      for (const edge of this.edges) {
        if (!edge.fromNode || !edge.toNode) continue;
        const active = (!this.query && !this.type) || (visible.has(edge.from.uri) && visible.has(edge.to.uri));
        context.strokeStyle = active ? "#4b5d56aa" : "#29332f55";
        context.lineWidth = active ? 1 : 0.6;
        context.beginPath();
        context.moveTo(edge.fromNode.x * box.width, edge.fromNode.y * box.height);
        context.lineTo(edge.toNode.x * box.width, edge.toNode.y * box.height);
        context.stroke();
      }
      for (const node of this.nodes) {
        const active = this.matches(node);
        const x = node.x * box.width;
        const y = node.y * box.height;
        context.globalAlpha = active ? 1 : 0.18;
        context.fillStyle = this.color(node.type);
        context.beginPath();
        context.arc(x, y, 5, 0, Math.PI * 2);
        context.fill();
        if (this.nodes.length < 55 && active) {
          context.fillStyle = "#d8d9d2";
          context.font = "11px ui-monospace, monospace";
          context.fillText(node.title, x + 9, y + 4);
        }
      }
      context.globalAlpha = 1;
    },

    pick(event) {
      const box = this.canvas.getBoundingClientRect();
      let closest = null;
      let distance = 15;
      for (const node of this.nodes) {
        const current = Math.hypot(
          node.x * box.width - (event.clientX - box.left),
          node.y * box.height - (event.clientY - box.top),
        );
        if (current < distance) {
          closest = node;
          distance = current;
        }
      }
      if (closest) this.$store.app.openPage(closest.uri);
    },
  }));

  Alpine.data("pagesView", () => ({
    pages: [],
    query: "",
    type: "",
    loaded: false,
    error: "",

    init() {
      lazy(this, "pages", () => {
        if (!this.loaded) this.load();
      });
    },

    async load() {
      this.error = "";
      try {
        const data = await api("/api/v1/pages");
        this.pages = (data.pages || []).map((page) => ({
          title: "",
          type: "",
          description: "",
          uri: "",
          ...page,
        }));
        this.loaded = true;
      } catch (error) {
        this.error = error.message;
      }
    },

    types() {
      return [...new Set(this.pages.map((page) => page.type))].sort();
    },

    visible() {
      const query = this.query.toLowerCase();
      return this.pages.filter((page) => {
        const text = `${page.title} ${page.type} ${page.description || ""} ${page.uri}`.toLowerCase();
        return (!this.type || page.type === this.type) && (!query || text.includes(query));
      });
    },
  }));

  Alpine.data("conceptsView", () => ({
    types: [],
    records: [],
    selected: "",
    loading: false,
    loaded: false,
    error: "",

    init() {
      lazy(this, "concepts", () => {
        if (!this.loaded) this.load();
      });
    },

    async load() {
      this.error = "";
      try {
        const data = await api("/api/v1/concepts");
        this.types = (data.concept_types || []).map((concept) => ({
          type: "",
          description: "",
          ...concept,
        }));
        this.loaded = true;
      } catch (error) {
        this.error = error.message;
      }
    },

    retry() {
      if (!this.loaded) {
        this.load();
        return;
      }
      if (this.selected) {
        const type = this.selected;
        this.selected = "";
        this.choose(type);
      }
    },

    async choose(type) {
      if (this.loading || this.selected === type) return;
      this.selected = type;
      this.loading = true;
      this.error = "";
      try {
        const data = await api("/api/v1/concepts?type=" + encodeURIComponent(type));
        this.records = (data.concepts || []).map((record) => ({ uri: "", ...record }));
      } catch (error) {
        this.error = error.message;
        this.records = [];
      } finally {
        this.loading = false;
      }
    },
  }));

  Alpine.data("searchView", () => ({
    question: "",
    backend: "lexical",
    searching: false,
    result: null,
    error: "",

    init() {
      this.$watch("$store.app.route.name", () => this.consume());
      this.consume();
    },

    // consume picks up a question handed over by another view (dashboard).
    consume() {
      const question = this.$store.app.pendingQuestion;
      if (!question || this.$store.app.route.name !== "search") return;
      this.$store.app.pendingQuestion = "";
      this.question = question;
      this.run();
    },

    async run() {
      const question = this.question.trim();
      if (!question || this.searching) return;
      this.searching = true;
      this.error = "";
      this.result = null;
      try {
        const params = new URLSearchParams({ question, backend: this.backend });
        const result = await api("/api/v1/search?" + params);
        result.candidates = (result.candidates || []).map((candidate) => ({
          score: 0,
          type: "",
          description: "",
          uri: "",
          title: "",
          ...candidate,
        }));
        result.should_read = result.should_read || [];
        this.result = result;
      } catch (error) {
        this.error = error.message;
      } finally {
        this.searching = false;
      }
    },
  }));

  Alpine.data("vaultsView", () => ({
    vaults: [],
    loaded: false,
    error: "",

    init() {
      lazy(this, "vaults", () => {
        if (!this.loaded) this.load();
      });
    },

    async load() {
      this.error = "";
      try {
        const data = await api("/api/v1/vaults");
        this.vaults = (data.vaults || []).map((vault) => ({
          vault: "",
          kind: "",
          precedence: 0,
          ...vault,
        }));
        this.loaded = true;
      } catch (error) {
        this.error = error.message;
      }
    },
  }));

  Alpine.data("readerView", () => ({
    page: null,
    uri: "",
    raw: false,
    loading: false,
    error: "",
    controller: null,
    tab: "content",
    history: [],
    historyCursor: "",
    historyError: "",
    historyExpired: false,
    historyLoading: false,
    historyLoaded: false,
    picked: [],
    diff: null,
    diffError: "",
    diffLoading: false,

    init() {
      this.$watch("$store.app.route", (route) => {
        if (route.name === "page" && route.uri !== this.uri) this.load(route.uri);
      });
      const route = Alpine.store("app").route;
      if (route.name === "page") this.load(route.uri);
    },

    shorten: shortRevision,

    async load(uri) {
      if (this.controller) this.controller.abort();
      const controller = new AbortController();
      this.controller = controller;
      this.uri = uri;
      this.raw = false;
      this.page = null;
      this.error = "";
      this.loading = true;
      this.resetHistory();
      try {
        const page = await api("/api/v1/page?uri=" + encodeURIComponent(uri), {
          signal: controller.signal,
        });
        page.document = page.document || {};
        page.document.origin = page.document.origin || {};
        this.page = page;
        this.$nextTick(() => window.scrollTo(0, 0));
      } catch (error) {
        if (error.aborted) return;
        this.error = error.message;
      } finally {
        if (this.controller === controller) this.loading = false;
      }
    },

    resetHistory() {
      this.tab = "content";
      this.history = [];
      this.historyCursor = "";
      this.historyError = "";
      this.historyExpired = false;
      this.historyLoading = false;
      this.historyLoaded = false;
      this.picked = [];
      this.diff = null;
      this.diffError = "";
      this.diffLoading = false;
    },

    showHistory() {
      this.tab = "history";
      if (!this.historyLoaded && !this.historyLoading) this.loadHistory();
    },

    async loadHistory(cursor = "") {
      this.historyLoading = true;
      this.historyError = "";
      this.historyExpired = false;
      const uri = this.uri;
      try {
        const params = new URLSearchParams({ uri, limit: "20" });
        if (cursor) params.set("cursor", cursor);
        const result = await api("/api/v1/history?" + params);
        if (uri !== this.uri) return;
        const entries = (result.entries || []).map((entry) => ({
          classification: "changed",
          timestamp: "",
          actor: "",
          revision: "",
          ...entry,
        }));
        this.history = cursor ? this.history.concat(entries) : entries;
        this.historyCursor = result.next_cursor || "";
        this.historyLoaded = true;
      } catch (error) {
        if (uri !== this.uri) return;
        if (cursor && error.status === 410) {
          this.historyCursor = "";
          this.historyExpired = true;
        } else {
          this.historyError = error.message;
        }
      } finally {
        if (uri === this.uri) this.historyLoading = false;
      }
    },

    restartHistory() {
      this.history = [];
      this.loadHistory();
    },

    togglePick(entry) {
      this.diffError = "";
      const index = this.picked.indexOf(entry.revision);
      if (index >= 0) {
        this.picked.splice(index, 1);
        this.diff = null;
        return;
      }
      this.picked.push(entry.revision);
      if (this.picked.length > 2) this.picked.shift();
      if (this.picked.length === 2) this.loadDiff();
    },

    async loadDiff() {
      if (this.picked.length !== 2) return;
      // history is newest-first, so the later list entry is the older revision.
      const ordered = this.history.filter((entry) => this.picked.includes(entry.revision));
      const from = ordered.length === 2 ? ordered[1].revision : this.picked[0];
      const to = ordered.length === 2 ? ordered[0].revision : this.picked[1];
      this.diffLoading = true;
      this.diffError = "";
      this.diff = null;
      const uri = this.uri;
      try {
        const params = new URLSearchParams({ uri, from, to });
        const result = await api("/api/v1/diff?" + params);
        if (uri !== this.uri) return;
        this.diff = result;
      } catch (error) {
        if (uri !== this.uri) return;
        this.diffError =
          error.status === 404
            ? "That revision is no longer available. Pick another pair."
            : error.message;
      } finally {
        if (uri === this.uri) this.diffLoading = false;
      }
    },

    // follow routes gnosis links inside rendered Markdown back into the
    // reader and opens external links in a new tab.
    follow(event) {
      const anchor = event.target.closest("a[href]");
      if (!anchor) return;
      const href = anchor.getAttribute("href");
      if (href.startsWith("gnosis://")) {
        event.preventDefault();
        this.$store.app.openPage(href);
      } else if (/^https?:\/\//.test(href)) {
        anchor.target = "_blank";
        anchor.rel = "noopener";
      }
    },
  }));
});

window.addEventListener("hashchange", () => {
  Alpine.store("app").route = parseHash();
});

Alpine.start();
