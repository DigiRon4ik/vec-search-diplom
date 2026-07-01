# План: Микросервисный маркетплейс с семантическим (векторным) поиском

## Статус: РЕАЛИЗОВАНО ✓

Прототип полностью реализован. Этот план отражает итоговое состояние системы.

## Context

Цель — демонстрационный прототип (дипломный) маркетплейса, показывающий **семантический поиск** по
каталогу товаров на русском языке. Вместо поиска по ключевым словам запрос пользователя
преобразуется в вектор-эмбеддинг, и по индексу HNSW (USearch) находятся семантически близкие товары.

Прототип демонстрирует: микросервисную архитектуру, межсервисное взаимодействие по HTTP/REST,
контейнеризацию через docker-compose, кросс-язычный поиск и бесконечную прокрутку каталога.

### Зафиксированные решения
- **Эмбеддинги**: отдельный Python-микросервис на FastAPI + `sentence-transformers`,
  модель `paraphrase-multilingual-MiniLM-L12-v2` (мультиязычная, размерность **384**, хороша для русского).
  Нормализация `normalize_embeddings=True`.
- **Хранилище каталога**: PostgreSQL (отдельный контейнер).
- **Тестовые данные**: синтетический каталог **10 000 товаров** (5000 RU + 5000 EN), генерируется скриптом.
  Мультиязычная модель — кросс-язычный поиск (RU-запрос находит EN-товары и наоборот).
- **Векторный поиск**: USearch через Go-биндинги `github.com/unum-cloud/usearch/golang` (cgo + C++).
  Метрика **InnerProduct** (не Cosine). Собирается из исходников в Docker.
- Остальные сервисы (каталог, поиск, шлюз) — на Go.

## Архитектура

```
                   ┌─────────────┐
   браузер ──────▶ │   gateway   │ (Go :8080)  + раздача статики фронтенда
                   └──────┬──────┘
        ┌─────────────────┼─────────────────────┐
        ▼                 ▼                       ▼
 ┌────────────┐   ┌────────────────┐     ┌──────────────┐
 │  catalog   │   │   embedding    │     │    search    │
 │ (Go :8081) │   │ (Py FastAPI    │     │ (Go :8083    │
 │            │   │     :8082)     │     │  USearch)    │
 └─────┬──────┘   └────────────────┘     └──────────────┘
       ▼                  ▲                     │
 ┌────────────┐          └── индексация ────────┘
 │ PostgreSQL │            (search дергает embedding + catalog)
 └────────────┘
```

### Потоки
**Индексация** (при старте search-service и/или по запросу `POST /index`):
search → `catalog` (получить все товары) → для каждого товара `embedding` (текст `name + category + description` → вектор 384) → добавить в USearch-индекс с ключом = product id.

**Поиск** (запрос пользователя):
браузер → `gateway POST /api/search {query}` → `embedding` (вектор запроса) → `search POST /search {vector,k}` (вернёт `[{id, distance}]`) → `gateway` обогащает результат данными товара из `catalog` → отдаёт фронтенду.

Такое разделение делает `search` чистым векторным сервисом (принимает/возвращает векторы и id), а `gateway` — оркестратором.

## Модель данных

```go
// services/catalog/internal/domain: сущность товара
type Product struct {
    ID          int64  `json:"id"`
    Name        string `json:"name"`
    Category    string `json:"category"`
    Description string `json:"description"`
    Lang        string `json:"lang"` // "ru" | "en" — язык карточки товара
}
```

PostgreSQL: таблица `products(id BIGSERIAL PK, name TEXT, category TEXT, description TEXT, lang TEXT)`, миграция при старте.

## Структура репозитория

Монорепозиторий; каждый Go-сервис организован по принципам **чистой архитектуры** (слои:
domain → usecase/service → adapters: repository/clients + transport/http; зависимости направлены внутрь,
к domain). Точка входа `cmd/server/main.go` собирает зависимости (composition root).

```
dip-vec-search/
  docker-compose.yml
  README.md
  .env.example
  go.work                                  # объединяет Go-модули

  services/
    catalog/
      go.mod
      Dockerfile
      cmd/server/main.go                   # composition root: wiring зависимостей
      internal/
        domain/        product.go          # сущность Product + интерфейс ProductRepository
        usecase/       product_service.go  # бизнес-логика (список, получить, создать, сидинг)
        repository/    postgres.go         # реализация ProductRepository на pgx
        transport/http/ handler.go router.go  # HTTP-обработчики, маршруты
      config/          config.go           # чтение env

    search/
      go.mod
      Dockerfile
      cmd/server/main.go
      internal/
        domain/        search.go           # типы Neighbor/Query + интерфейсы VectorIndex, Embedder, Catalog
        usecase/       index_service.go search_service.go  # построение индекса, kNN
        adapters/
          usearch/     index.go            # реализация VectorIndex на USearch
          embedding/   client.go           # HTTP-клиент embedding-сервиса
          catalog/     client.go           # HTTP-клиент catalog-сервиса
        transport/http/ handler.go router.go
      config/          config.go

    gateway/
      go.mod
      Dockerfile
      cmd/server/main.go
      internal/
        domain/        search.go           # агрегированный результат (Product + score)
        usecase/       search_service.go   # оркестрация embedding→search→catalog
        adapters/
          embedding/   client.go
          search/      client.go
          catalog/     client.go
        transport/http/ handler.go router.go  # /api/*, раздача статики
      config/          config.go

    embedding/                              # Python, слоистая структура
      app/
        main.py                            # FastAPI app, роуты
        core/embedder.py                   # загрузка модели + батч-эмбеддинг
        schemas.py                         # pydantic-модели запрос/ответ
      requirements.txt
      Dockerfile

  frontend/    index.html  app.js  styles.css   # раздаётся gateway, без сборщика

  data/
    catalog_seed.json                      # 1000 товаров (RU + EN), сгенерировано
  scripts/
    gen_catalog.py                         # генератор синтетического каталога (RU + EN)
  eval/
    queries.json                           # тест-набор запросов (RU + EN) + ожидаемые категории/id
    eval.py                                # метрики качества (precision@k) и latency
```

## Сервисы — детали реализации

### 1. catalog-service (Go, :8081)
- REST API:
  - `GET /products` — список (с пагинацией опционально)
  - `GET /products/{id}` — один товар
  - `POST /products` — добавить (для наполнения)
  - `GET /health`
- PostgreSQL через `database/sql` + `github.com/jackc/pgx/v5/stdlib` (или `lib/pq`).
- Наполнение: при старте, если таблица пуста, загрузить `data/catalog_seed.json`.

### 2. embedding-service (Python FastAPI, :8082)
- `POST /embed` — тело `{"texts": ["..."]}`, ответ `{"vectors": [[...384...]]}`.
- `GET /health`.
- Модель `sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2` загружается один раз при старте.
  Нормализация эмбеддингов (`normalize_embeddings=True`) под косинусную метрику.
- Поддержка батча для ускорения индексации.

### 3. search-service (Go, :8083) — USearch/HNSW
- USearch-индекс: метрика **`InnerProduct`** (не Cosine), размерность 384, F32, Connectivity=32.
  ```go
  usearch.IndexConfig{Metric: usearch.InnerProduct, Dimensions: 384, Quantization: usearch.F32,
      Connectivity: 32, ExpansionAdd: 128, ExpansionSearch: 64}
  ```
- **Score normalization**: `distance` от USearch при IP ∈ [0,2], где `ip = 1 - distance`, `score = (ip+1)/2` → [0,1].
  Идентичный товар: distance=0 → score=1.0.
- **`VectorIndex` интерфейс** включает метод `Reserve(capacity int) error`.
- **Порядок при построении**: `Reset()` → `Reserve(n)` → `Add(...)` (Reserve обязателен для IP-метрики).
- **`GetAll()` в catalog-клиенте**: пагинирует страницами по 500 товаров до исчерпания (не один запрос).
  `services/search/internal/adapters/catalog/client.go`
- Ответ `/search`: `{"results":[{"id":N,"distance":D,"score":S}]}` — поле `score` добавлено.
- Эндпоинты:
  - `POST /index` — (пере)построить индекс: тянет все 10 000 товаров из catalog, эмбеддинги из embedding.
  - `POST /search` — kNN-поиск.
  - `GET /health`, `GET /stats`.
- При старте — автоматически вызвать построение индекса (с ретраями).
- Ключи USearch = product id (uint64).

### 4. gateway (Go, :8080)
- `POST /api/search` — тело `{"query":"...", "k":10}`: оркестрирует embedding → search → catalog-hydration;
  возвращает `[{product, distance, score}]` (поле `score` ∈ [0,1]).
- `POST /api/reindex` — проксирует на `search POST /index`.
- `GET /api/products` — проксирует на catalog (поддерживает `?limit`, `?offset`, `?category`, `?ids=1,2,3`).
- `GET /api/categories` — проксирует на catalog.
- Раздаёт статику фронтенда (`/`), `GET /health`.
- HTTP-клиенты к сервисам с таймаутами; адреса сервисов из env.

### 5. frontend (vanilla HTML/CSS/JS)
- Боковая панель категорий с иконками (эмодзи); кнопка «Все» — сброс фильтра.
- Каталог: 3-колоночная сетка карточек, бесконечная прокрутка (IntersectionObserver на `#sentinel`),
  детерминированный порядок без дублей (`ORDER BY md5(id::text)` в PostgreSQL).
- Поиск: запрос k=100, фильтр `score > 0.5`, пагинация по 24 карточки через тот же sentinel.
  Первая карточка — наиболее близкая (score=1.0 = идентичная).
- Фильтр по категории работает и в режиме каталога, и в режиме поиска.
- Без сборщика — статические файлы `frontend/index.html`, `app.js`, `styles.css`.
- Ключевая деталь: `<div id="sentinel">` (1px, всегда в DOM) — отдельный от `#loader`,
  чтобы observer не терял цель при `display:none`.

## Синтетический каталог (`scripts/gen_catalog.py`)
- Генерирует **10 000 товаров** (5000 RU id 1–5000, 5000 EN id 5001–10000).
- Пулы шаблонов: 50 брендов × язык, 30 шаблонов названий × категорию × язык,
  10 шаблонов описаний × категорию × язык, 30 книжных тем.
- Итог: 8296 уникальных названий из 10 000 записей.
- Поскольку модель эмбеддингов мультиязычная — кросс-язычный поиск:
  «wireless headphones» находит «Беспроводные наушники» (cos sim ≈ 0.85).
- Результат: `data/catalog_seed.json`.

## Контейнеризация (`docker-compose.yml`)
Сервисы: `postgres`, `catalog`, `embedding`, `search`, `gateway`.
- `depends_on` + healthcheck'и: catalog ждёт postgres; search ждёт catalog и embedding; gateway ждёт search и catalog.
- Адреса сервисов передаются через env (`CATALOG_URL`, `EMBEDDING_URL`, `SEARCH_URL`).
- Порт наружу — только gateway (`8080:8080`); остальные во внутренней сети compose.
- Тома: `hf_cache` (модель embedding), `pgdata` (PostgreSQL).
- `gateway` использует `additional_contexts: frontend: ./frontend` для COPY статики.
- Go-сервисы: multi-stage Dockerfile. Для `search` — USearch C-библиотека собирается из исходников
  (`cmake -DUSEARCH_BUILD_LIB_C=ON`) на этапе сборки.
- Все `COPY go.mod go.sum* ./` — `go.sum*` (со звёздочкой), т.к. gateway не имеет go.sum.

## Оценка качества и скорости (`eval/`)
- `eval/queries.json`: ~15–20 запросов (часть на русском, часть на английском, в т.ч. кросс-язычные) с ожидаемыми релевантными категориями/id товаров.
- `eval/eval.py`: для каждого запроса дергает `gateway /api/search`, считает **precision@k / recall@k**
  по ожидаемым категориям, и замеряет **latency** (p50/p95) по эндпоинту поиска.
- Вывод сводной таблицы в консоль (и опц. в `eval/report.md`).

## Итоговый порядок реализации (выполнено)
1. ✓ Каркас репозитория, `go.work`, `docker-compose.yml`.
2. ✓ `scripts/gen_catalog.py` → `data/catalog_seed.json` (10 000 товаров).
3. ✓ catalog-service + миграция + сидинг; `GET /products?limit&offset&category&ids`.
4. ✓ embedding-service (FastAPI + `paraphrase-multilingual-MiniLM-L12-v2`).
5. ✓ search-service + USearch InnerProduct + score normalization + Reserve().
6. ✓ gateway + проксирование /api/products, /api/categories, /api/search, /api/reindex.
7. ✓ frontend: 3-col grid, категории, бесконечная прокрутка, фильтрация.
8. ✓ docker-compose: все сервисы, healthcheck'и, тома hf_cache/pgdata.
9. ✓ eval-скрипт.

## Verification (проверено end-to-end)
1. `docker compose up --build` — 5 сервисов + postgres, healthcheck'и зелёные.
2. `curl localhost:8080/health` и `curl localhost:8083/stats` — индекс 10 000 векторов.
3. `http://localhost:8080` — каталог с бесконечной прокруткой без дублей; поиск по категории.
4. Семантический запрос «наушники» → score top-1 ≈ 0.86; «wireless headphones» → находит RU-товары.
5. Фильтр: `score > 0.5` отсекает нерелевантные результаты (отрицательное косинусное сходство).
6. `python eval/eval.py` — precision@k и latency p50/p95.
7. `POST /api/reindex` — переиндексация без перезапуска.

## Известные особенности реализации
- **USearch Reserve**: для метрики InnerProduct `Reserve(n)` обязателен до первого `Add()`, иначе паника.
- **Sentinel vs loader**: `IntersectionObserver` наблюдает `#sentinel` (1px, всегда в DOM),
  а не `#loader` (который скрывается через `display:none` и теряется observer'ом).
- **`applySearchCategoryFilter()`** вызывается после `finally`-блока (вне try/catch),
  чтобы не конфликтовать с `loading=true` внутри `doSearch()`.
- **go.sum optional**: gateway не имеет внешних зависимостей → `go.sum` отсутствует;
  все Dockerfile используют `COPY go.mod go.sum* ./` (glob, не жёсткое имя).
- **Размер модели**: первый старт embedding качает ~470 МБ (кэш в томе `hf_cache`).
