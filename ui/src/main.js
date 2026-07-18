import Alpine from "alpinejs";
import "./styles.css";

// api fetches one JSON endpoint and surfaces server error messages.
const api = async (path) => {
  const response = await fetch(path);
  const data = await response.json();
  if (!response.ok) throw new Error(data.error || `HTTP ${response.status}`);
  return data;
};

const views = ["graph", "pages", "concepts", "search", "vaults"];

const parseHash = () => {
  const hash = location.hash.replace(/^#\/?/, "");
  const [name = "", ...rest] = hash.split("/");
  if (name === "page" && rest.length > 0) {
    return { name: "page", uri: decodeURIComponent(rest.join("/")) };
  }
  return { name: views.includes(name) ? name : "graph" };
};

// lazy runs load the first time the named view becomes active.
const lazy = (component, view, load) => {
  component.$watch("$store.app.route.name", (name) => {
    if (name === view) load();
  });
  if (Alpine.store("app").route.name === view) load();
};

document.addEventListener("alpine:init", () => {
  Alpine.store("app", {
    route: parseHash(),
    is(name) {
      return this.route.name === name;
    },
    openPage(uri) {
      location.hash = "#/page/" + encodeURIComponent(uri);
    },
  });

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
      try {
        const graph = await api("/api/v1/graph");
        const nodes = graph.nodes || [];
        const count = Math.max(1, nodes.length);
        this.nodes = nodes.map((node, index) => {
          const angle = index * 2.399963229728653;
          const radius = 0.08 + 0.34 * Math.sqrt((index + 1) / count);
          return { ...node, x: 0.5 + Math.cos(angle) * radius, y: 0.5 + Math.sin(angle) * radius };
        });
        const byURI = new Map(this.nodes.map((node) => [node.uri, node]));
        this.edges = (graph.edges || []).map((edge) => ({
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
        if (this.loaded) return;
        api("/api/v1/pages")
          .then((data) => {
            this.pages = data.pages || [];
            this.loaded = true;
          })
          .catch((error) => {
            this.error = error.message;
          });
      });
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
        if (this.loaded) return;
        api("/api/v1/concepts")
          .then((data) => {
            this.types = data.concept_types || [];
            this.loaded = true;
          })
          .catch((error) => {
            this.error = error.message;
          });
      });
    },

    async choose(type) {
      if (this.loading || this.selected === type) return;
      this.selected = type;
      this.loading = true;
      this.error = "";
      try {
        const data = await api("/api/v1/concepts?type=" + encodeURIComponent(type));
        this.records = data.concepts || [];
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

    async run() {
      const question = this.question.trim();
      if (!question || this.searching) return;
      this.searching = true;
      this.error = "";
      this.result = null;
      try {
        const params = new URLSearchParams({ question, backend: this.backend });
        const result = await api("/api/v1/search?" + params);
        result.candidates = result.candidates || [];
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
        if (this.loaded) return;
        api("/api/v1/vaults")
          .then((data) => {
            this.vaults = data.vaults || [];
            this.loaded = true;
          })
          .catch((error) => {
            this.error = error.message;
          });
      });
    },
  }));

  Alpine.data("readerView", () => ({
    page: null,
    uri: "",
    raw: false,
    loading: false,
    error: "",

    init() {
      this.$watch("$store.app.route", (route) => {
        if (route.name === "page" && route.uri !== this.uri) this.load(route.uri);
      });
      const route = Alpine.store("app").route;
      if (route.name === "page") this.load(route.uri);
    },

    shorten(revision) {
      return (revision || "").replace(/^sha256:/, "").slice(0, 12);
    },

    async load(uri) {
      this.uri = uri;
      this.raw = false;
      this.page = null;
      this.error = "";
      this.loading = true;
      try {
        this.page = await api("/api/v1/page?uri=" + encodeURIComponent(uri));
        this.$nextTick(() => window.scrollTo(0, 0));
      } catch (error) {
        this.error = error.message;
      } finally {
        this.loading = false;
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
