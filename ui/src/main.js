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

const views = ["dashboard", "pages", "concepts", "search", "procedures", "vaults"];

// maxHoodNodes is the hard cap for the reader's bounded ego-graph; beyond it
// the UI asks for narrower direction or relation filters instead of rendering.
const maxHoodNodes = 300;

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

// graphCanvas is the bounded ego-graph renderer retained from the retired
// global graph view: golden-angle seeding, a short force settle, canvas draw,
// and click-to-open picking. It renders the reader's neighborhood nodes.
const graphCanvas = () => ({
  hoodCanvas: null,
  hoodContext: null,
  hoodFrames: 0,

  color(value) {
    let hash = 0;
    for (const character of value) hash = (hash * 31 + character.charCodeAt(0)) | 0;
    return `hsl(${Math.abs(hash) % 360} 48% 64%)`;
  },

  resizeGraph() {
    const canvas = this.$refs.hoodCanvas;
    if (!canvas || !canvas.clientWidth) return;
    this.hoodCanvas = canvas;
    const box = canvas.getBoundingClientRect();
    const ratio = window.devicePixelRatio || 1;
    canvas.width = Math.max(1, Math.floor(box.width * ratio));
    canvas.height = Math.max(1, Math.floor(box.height * ratio));
    this.hoodContext = canvas.getContext("2d");
    this.hoodContext.setTransform(ratio, 0, 0, ratio, 0, 0);
    this.drawGraph();
  },

  settleGraph() {
    if (!this.hoodNodes.length || this.hoodFrames > 180) {
      this.drawGraph();
      return;
    }
    const nodes = this.hoodNodes;
    for (let i = 0; i < nodes.length; i++) {
      nodes[i].dx = 0;
      nodes[i].dy = 0;
      for (let j = i + 1; j < nodes.length; j++) {
        const x = nodes[j].x - nodes[i].x;
        const y = nodes[j].y - nodes[i].y;
        const distance = Math.max(0.002, x * x + y * y);
        const force = Math.min(0.00012 / distance, 0.012);
        nodes[i].dx -= x * force;
        nodes[i].dy -= y * force;
        nodes[j].dx += x * force;
        nodes[j].dy += y * force;
      }
    }
    for (const edge of this.hoodEdges) {
      if (!edge.fromNode || !edge.toNode) continue;
      const x = edge.toNode.x - edge.fromNode.x;
      const y = edge.toNode.y - edge.fromNode.y;
      const force = Math.max(0, Math.hypot(x, y) - 0.16) * 0.018;
      edge.fromNode.dx += x * force;
      edge.fromNode.dy += y * force;
      edge.toNode.dx -= x * force;
      edge.toNode.dy -= y * force;
    }
    for (const node of nodes) {
      node.dx += (0.5 - node.x) * 0.00035;
      node.dy += (0.5 - node.y) * 0.00035;
      node.x = Math.min(0.94, Math.max(0.06, node.x + node.dx));
      node.y = Math.min(0.91, Math.max(0.09, node.y + node.dy));
    }
    this.hoodFrames++;
    this.drawGraph();
    requestAnimationFrame(() => this.settleGraph());
  },

  drawGraph() {
    if (!this.hoodContext || !this.hoodCanvas) return;
    const box = this.hoodCanvas.getBoundingClientRect();
    const context = this.hoodContext;
    context.clearRect(0, 0, box.width, box.height);
    context.strokeStyle = "#4b5d56aa";
    context.lineWidth = 1;
    for (const edge of this.hoodEdges) {
      if (!edge.fromNode || !edge.toNode) continue;
      context.beginPath();
      context.moveTo(edge.fromNode.x * box.width, edge.fromNode.y * box.height);
      context.lineTo(edge.toNode.x * box.width, edge.toNode.y * box.height);
      context.stroke();
    }
    for (const node of this.hoodNodes) {
      const x = node.x * box.width;
      const y = node.y * box.height;
      context.fillStyle = this.color(node.type);
      context.beginPath();
      context.arc(x, y, 5, 0, Math.PI * 2);
      context.fill();
      if (node.uri === this.uri) {
        context.strokeStyle = "#e8b86d";
        context.lineWidth = 2;
        context.beginPath();
        context.arc(x, y, 8, 0, Math.PI * 2);
        context.stroke();
      }
      if (this.hoodNodes.length < 55) {
        context.fillStyle = "#d8d9d2";
        context.font = "11px ui-monospace, monospace";
        context.fillText(node.title, x + 9, y + 4);
      }
    }
  },

  pickGraph(event) {
    if (!this.hoodCanvas) return;
    const box = this.hoodCanvas.getBoundingClientRect();
    let closest = null;
    let distance = 15;
    for (const node of this.hoodNodes) {
      const current = Math.hypot(
        node.x * box.width - (event.clientX - box.left),
        node.y * box.height - (event.clientY - box.top),
      );
      if (current < distance) {
        closest = node;
        distance = current;
      }
    }
    if (closest && closest.uri !== this.uri) this.$store.app.openPage(closest.uri);
  },
});

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
        // the pages endpoint is paginated: the exact count comes from total,
        // and the per-type breakdown is only honest when one page holds all.
        const list = (pages.pages || []).map((page) => ({ type: "", ...page }));
        const byType = {};
        if (!pages.next_cursor) {
          for (const page of list) {
            const type = page.type || "unknown";
            byType[type] = (byType[type] || 0) + 1;
          }
        }
        this.pulse = {
          pages: pages.total ?? list.length,
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

  Alpine.data("pagesView", () => ({
    pages: [],
    total: 0,
    cursor: "",
    query: "",
    type: "",
    types: [],
    loading: false,
    loaded: false,
    error: "",
    expired: false,

    init() {
      // query/type edits restart pagination against the server (debounced by
      // x-model on the input) instead of filtering a client-side dump.
      this.$watch("query", () => this.restart());
      this.$watch("type", () => this.restart());
      lazy(this, "pages", () => {
        if (!this.loaded) this.load();
        if (!this.types.length) this.loadTypes();
      });
    },

    // loadTypes feeds the type filter from the bounded concept-type catalog.
    async loadTypes() {
      try {
        const data = await api("/api/v1/concepts");
        this.types = (data.concept_types || []).map((concept) => concept.type).sort();
      } catch (error) {
        // a failed type catalog only limits the filter; the list still loads.
      }
    },

    restart() {
      if (!this.loaded) return;
      this.pages = [];
      this.cursor = "";
      this.total = 0;
      this.expired = false;
      this.error = "";
      this.load();
    },

    async load(cursor = "") {
      if (this.loading) return;
      this.loading = true;
      this.error = "";
      this.expired = false;
      try {
        const params = new URLSearchParams();
        if (this.query.trim()) params.set("q", this.query.trim());
        if (this.type) params.set("type", this.type);
        if (cursor) params.set("cursor", cursor);
        const suffix = params.toString();
        const data = await api("/api/v1/pages" + (suffix ? "?" + suffix : ""));
        const batch = (data.pages || []).map((page) => ({
          title: "",
          type: "",
          description: "",
          uri: "",
          ...page,
        }));
        this.pages = cursor ? this.pages.concat(batch) : batch;
        this.total = data.total ?? this.pages.length;
        this.cursor = data.next_cursor || "";
        this.loaded = true;
      } catch (error) {
        if (cursor && error.status === 410) {
          this.cursor = "";
          this.expired = true;
        } else {
          this.error = error.message;
        }
      } finally {
        this.loading = false;
      }
    },

    restartExpired() {
      this.pages = [];
      this.expired = false;
      this.load();
    },
  }));

  Alpine.data("proceduresView", () => ({
    procedures: [],
    selected: "",
    contract: null,
    loading: false,
    contractLoading: false,
    loaded: false,
    error: "",
    contractError: "",

    init() {
      lazy(this, "procedures", () => {
        if (!this.loaded) this.load();
      });
    },

    short: shortRevision,

    async load() {
      this.error = "";
      try {
        const data = await api("/api/v1/procedures");
        this.procedures = (data.procedures || []).map((procedure) => ({
          title: "",
          description: "",
          tags: [],
          trust: {},
          ...procedure,
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
        const uri = this.selected;
        this.selected = "";
        this.choose(uri);
      }
    },

    // choose reads one exact execution contract; the view never executes it.
    async choose(uri) {
      if (this.contractLoading || this.selected === uri) return;
      this.selected = uri;
      this.contract = null;
      this.contractError = "";
      this.contractLoading = true;
      try {
        const data = await api("/api/v1/procedures?uri=" + encodeURIComponent(uri));
        this.contract = data.procedure;
      } catch (error) {
        this.contractError = error.message;
      } finally {
        this.contractLoading = false;
      }
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
    ...graphCanvas(),
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
    hoodDirection: "both",
    hoodRelation: "",
    hoodDepth: "1",
    hoodNodes: [],
    hoodEdges: [],
    hoodLoading: false,
    hoodLoaded: false,
    hoodError: "",
    hoodCapped: false,
    hoodTruncated: false,
    pathTarget: "",
    pathDepth: "3",
    pathResult: null,
    pathError: "",
    pathLoading: false,

    init() {
      this.$watch("$store.app.route", (route) => {
        if (route.name === "page" && route.uri !== this.uri) this.load(route.uri);
      });
      window.addEventListener("resize", () => this.resizeGraph());
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
      this.resetPanels();
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

    resetPanels() {
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
      this.hoodNodes = [];
      this.hoodEdges = [];
      this.hoodLoading = false;
      this.hoodLoaded = false;
      this.hoodError = "";
      this.hoodCapped = false;
      this.hoodTruncated = false;
      this.hoodFrames = 0;
      this.pathTarget = "";
      this.pathResult = null;
      this.pathError = "";
      this.pathLoading = false;
    },

    showHistory() {
      this.tab = "history";
      if (!this.historyLoaded && !this.historyLoading) this.loadHistory();
    },

    showNeighborhood() {
      this.tab = "neighborhood";
      this.$nextTick(() => this.resizeGraph());
      if (!this.hoodLoaded && !this.hoodLoading) this.loadNeighborhood();
    },

    // loadNeighborhood walks the bounded ego-graph around the open page
    // through the traversal API: one hop at depth 1, a frontier expansion at
    // depth 2, always stopping at the hard node cap.
    async loadNeighborhood() {
      this.hoodLoading = true;
      this.hoodError = "";
      this.hoodCapped = false;
      this.hoodTruncated = false;
      const uri = this.uri;
      const relations = this.hoodRelation
        .split(",")
        .map((relation) => relation.trim())
        .filter(Boolean);
      const params = (target) => {
        const search = new URLSearchParams({
          uri: target,
          direction: this.hoodDirection,
          limit: "100",
        });
        for (const relation of relations) search.append("relation", relation);
        return search;
      };
      const nodes = new Map();
      const edges = new Map();
      const capped = () => nodes.size >= maxHoodNodes;
      const addEdge = (edge) => {
        edges.set(`${edge.from.uri} ${edge.relation} ${edge.to.uri}`, edge);
        for (const endpoint of [edge.from, edge.to]) {
          if (!nodes.has(endpoint.uri) && !capped()) nodes.set(endpoint.uri, endpoint);
        }
      };
      try {
        const first = await api("/api/v1/graph/neighbors?" + params(uri));
        if (uri !== this.uri) return;
        nodes.set(first.node.uri, first.node);
        for (const edge of first.edges || []) addEdge(edge);
        let truncated = first.truncated;
        if (Number(this.hoodDepth) > 1) {
          const frontier = [...nodes.keys()].filter((node) => node !== uri);
          for (const next of frontier) {
            if (capped()) break;
            const result = await api("/api/v1/graph/neighbors?" + params(next));
            if (uri !== this.uri) return;
            if (result.truncated) truncated = true;
            for (const edge of result.edges || []) addEdge(edge);
          }
        }
        this.hoodCapped = capped();
        this.hoodTruncated = truncated;
        const count = Math.max(1, nodes.size);
        this.hoodNodes = [...nodes.values()].map((node, index) => {
          const normalized = { type: "", title: "", uri: "", ...node };
          normalized.title = normalized.title || normalized.uri;
          const angle = index * 2.399963229728653;
          const radius = index === 0 ? 0 : 0.08 + 0.34 * Math.sqrt((index + 1) / count);
          return { ...normalized, x: 0.5 + Math.cos(angle) * radius, y: 0.5 + Math.sin(angle) * radius };
        });
        const byURI = new Map(this.hoodNodes.map((node) => [node.uri, node]));
        this.hoodEdges = [...edges.values()]
          .map((edge) => ({
            ...edge,
            fromNode: byURI.get(edge.from.uri),
            toNode: byURI.get(edge.to.uri),
          }))
          .filter((edge) => edge.fromNode && edge.toNode);
        this.hoodLoaded = true;
        this.hoodFrames = 0;
        this.$nextTick(() => {
          this.resizeGraph();
          this.settleGraph();
        });
      } catch (error) {
        if (uri !== this.uri) return;
        this.hoodError = error.message;
      } finally {
        if (uri === this.uri) this.hoodLoading = false;
      }
    },

    reloadNeighborhood() {
      if (!this.hoodLoaded && !this.hoodError) return;
      this.hoodLoaded = false;
      this.loadNeighborhood();
    },

    // findPath renders the bounded typed path from the open page to a target.
    async findPath() {
      const target = this.pathTarget.trim();
      if (!target || this.pathLoading) return;
      this.pathLoading = true;
      this.pathError = "";
      this.pathResult = null;
      const uri = this.uri;
      const depth = this.pathDepth;
      try {
        const params = new URLSearchParams({ uri, target, depth });
        const result = await api("/api/v1/graph/path?" + params);
        if (uri !== this.uri) return;
        switch (result.status) {
          case "found":
            this.pathResult = result;
            break;
          case "unknown_target":
            this.pathError = `No page exists with URI ${target}. Check the target and try again.`;
            break;
          case "unknown_source":
            this.pathError = "The open page is no longer in the vault.";
            break;
          case "depth_exceeded":
            this.pathError = `A path exists but needs more than ${depth} hops. Increase the depth bound.`;
            break;
          default:
            this.pathError = `No path reaches ${target} within ${depth} hops.`;
        }
      } catch (error) {
        if (uri !== this.uri) return;
        this.pathError = error.message;
      } finally {
        if (uri === this.uri) this.pathLoading = false;
      }
    },

    pathStepEdge(index) {
      return this.pathResult && this.pathResult.edges[index]
        ? this.pathResult.edges[index].relation
        : "";
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
