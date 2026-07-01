(() => {
  "use strict";

  const PAGE_SIZE = 24;
  const SEARCH_BATCH = 24;

  const $ = (s) => document.querySelector(s);
  const grid = $("#grid");
  const loader = $("#loader");
  const endMsg = $("#end-msg");
  const sentinel = $("#sentinel");
  const queryInput = $("#query");
  const clearBtn = $("#clear-btn");
  const catList = $("#cat-list");
  const pageTitle = $("#page-title");
  const totalBadge = $("#total-badge");

  let mode = "catalog"; // "catalog" | "search"
  let category = "";
  let catalogOffset = 0;
  let catalogTotal = 0;
  let loading = false;
  let allLoaded = false;

  let searchResultsFull = [];
  let searchResultsFiltered = [];
  let searchShown = 0;

  const CATEGORY_ICONS = {
    "Электроника": "💻", "Electronics": "💻",
    "Книги": "📚", "Books": "📚",
    "Одежда": "👕", "Clothing": "👕",
    "Спорт": "⚽", "Sports": "⚽",
    "Дом и кухня": "🏠", "Home & Kitchen": "🏠",
    "Красота": "💄", "Beauty": "💄",
    "Игрушки": "🧸", "Toys": "🧸",
    "Продукты": "🍎", "Food": "🍎",
    "Автотовары": "🚗", "Automotive": "🚗",
    "Сад и огород": "🌱", "Garden": "🌱",
  };

  const CARD_COLORS = [
    "#eef2ff", "#fef3f2", "#ecfdf5", "#fffbeb",
    "#f5f3ff", "#fdf2f8", "#f0fdfa", "#fefce8",
    "#f0f9ff", "#fff7ed",
  ];

  function cardColor(id) {
    return CARD_COLORS[id % CARD_COLORS.length];
  }

  function icon(cat) {
    return CATEGORY_ICONS[cat] || "📦";
  }

  function esc(s) {
    const d = document.createElement("div");
    d.textContent = s;
    return d.innerHTML;
  }

  // ── Cards ──

  function productCard(p, score) {
    const card = document.createElement("div");
    card.className = "card";
    const sim = score != null
      ? `<span class="card-score">${score.toFixed(2)} sim</span>`
      : "";
    card.innerHTML = `
      <div class="card-img" style="background:${cardColor(p.id)}">
        ${icon(p.category)}
      </div>
      <div class="card-body">
        <div class="card-cat">${esc(p.category)}</div>
        <div class="card-name">${esc(p.name)}</div>
        <div class="card-desc">${esc(p.description)}</div>
      </div>
      <div class="card-footer">
        <span class="card-lang">${p.lang === "en" ? "EN" : "RU"}</span>
        ${sim}
      </div>`;
    return card;
  }

  // ── Categories ──

  async function loadCategories() {
    try {
      const resp = await fetch("/api/categories");
      const data = await resp.json();
      for (const cat of data.categories || []) {
        const li = document.createElement("li");
        li.innerHTML = `<button class="cat-btn" data-category="${esc(cat)}">${icon(cat)} ${esc(cat)}</button>`;
        catList.appendChild(li);
      }
    } catch (e) {
      console.error("categories:", e);
    }
  }

  function activateCatBtn(cat) {
    catList.querySelectorAll(".cat-btn").forEach((b) => {
      b.classList.toggle("active", b.dataset.category === cat);
    });
  }

  // ── Catalog mode ──

  async function loadCatalogPage() {
    if (loading || allLoaded) return;
    loading = true;
    loader.classList.remove("hidden");
    endMsg.classList.add("hidden");

    try {
      const params = new URLSearchParams({
        limit: PAGE_SIZE,
        offset: catalogOffset,
      });
      if (category) params.set("category", category);

      const resp = await fetch("/api/products?" + params);
      const data = await resp.json();
      const products = data.products || [];
      catalogTotal = data.total || 0;
      totalBadge.textContent = `${catalogTotal} товаров`;

      for (const p of products) {
        grid.appendChild(productCard(p, null));
      }

      catalogOffset += products.length;
      if (products.length < PAGE_SIZE || catalogOffset >= catalogTotal) {
        allLoaded = true;
        endMsg.classList.remove("hidden");
      }
    } catch (e) {
      console.error("catalog:", e);
    } finally {
      loading = false;
      loader.classList.add("hidden");
    }
  }

  function resetCatalog() {
    grid.innerHTML = "";
    catalogOffset = 0;
    catalogTotal = 0;
    allLoaded = false;
    endMsg.classList.add("hidden");
    loadCatalogPage();
  }

  function enterCatalogMode() {
    mode = "catalog";
    searchResultsFull = [];
    searchResultsFiltered = [];
    searchShown = 0;
    pageTitle.textContent = category || "Каталог товаров";
    resetCatalog();
  }

  // ── Search mode ──

  async function doSearch(query) {
    mode = "search";
    grid.innerHTML = "";
    searchResultsFull = [];
    searchResultsFiltered = [];
    searchShown = 0;
    allLoaded = false;
    endMsg.classList.add("hidden");
    loading = true;
    loader.classList.remove("hidden");
    pageTitle.textContent = `Результаты: «${query}»`;
    totalBadge.textContent = "";

    try {
      const resp = await fetch("/api/search", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query, k: 100 }),
      });

      if (!resp.ok) throw new Error(await resp.text());

      const data = await resp.json();
      searchResultsFull = (data.results || [])
        .map((r) => ({
          product: r.product,
          score: r.score,
        }))
        .filter((r) => r.score > 0.5);

    } catch (e) {
      console.error("search:", e);
      pageTitle.textContent = "Ошибка поиска";
    } finally {
      loading = false;
      loader.classList.add("hidden");
    }

    if (mode === "search") applySearchCategoryFilter();
  }

  function applySearchCategoryFilter() {
    if (category) {
      searchResultsFiltered = searchResultsFull.filter(
        (r) => r.product.category === category
      );
    } else {
      searchResultsFiltered = searchResultsFull;
    }

    grid.innerHTML = "";
    searchShown = 0;
    allLoaded = false;
    endMsg.classList.add("hidden");

    totalBadge.textContent = `${searchResultsFiltered.length} найдено`;
    showSearchPage();
  }

  function showSearchPage() {
    if (loading || allLoaded) return;
    const end = Math.min(searchShown + SEARCH_BATCH, searchResultsFiltered.length);
    for (let i = searchShown; i < end; i++) {
      const { product, score } = searchResultsFiltered[i];
      grid.appendChild(productCard(product, score));
    }
    searchShown = end;
    if (searchShown >= searchResultsFiltered.length) {
      allLoaded = true;
      if (searchResultsFiltered.length > 0) endMsg.classList.remove("hidden");
    }
  }

  // ── Infinite scroll via sentinel ──

  const observer = new IntersectionObserver(
    (entries) => {
      if (!entries[0].isIntersecting || loading || allLoaded) return;
      if (mode === "catalog") loadCatalogPage();
      else showSearchPage();
    },
    { rootMargin: "600px" }
  );
  observer.observe(sentinel);

  // ── Events ──

  $("#search-form").addEventListener("submit", (e) => {
    e.preventDefault();
    const q = queryInput.value.trim();
    if (q) doSearch(q);
  });

  queryInput.addEventListener("input", () => {
    clearBtn.classList.toggle("hidden", !queryInput.value);
  });

  clearBtn.addEventListener("click", () => {
    queryInput.value = "";
    clearBtn.classList.add("hidden");
    category = "";
    activateCatBtn("");
    enterCatalogMode();
  });

  $("#logo").addEventListener("click", (e) => {
    e.preventDefault();
    queryInput.value = "";
    clearBtn.classList.add("hidden");
    category = "";
    activateCatBtn("");
    enterCatalogMode();
  });

  catList.addEventListener("click", (e) => {
    const btn = e.target.closest(".cat-btn");
    if (!btn) return;
    category = btn.dataset.category;
    activateCatBtn(category);

    if (mode === "search") {
      applySearchCategoryFilter();
    } else {
      pageTitle.textContent = category || "Каталог товаров";
      resetCatalog();
    }
  });

  // ── Init ──

  loadCategories();
  loadCatalogPage();
})();
