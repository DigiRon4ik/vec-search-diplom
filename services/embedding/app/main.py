from contextlib import asynccontextmanager

from fastapi import FastAPI

from app.core.embedder import Embedder
from app.schemas import EmbedRequest, EmbedResponse

embedder: Embedder | None = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    global embedder
    embedder = Embedder()
    print(f"Model loaded, dimension={embedder.dimension}")
    yield


app = FastAPI(title="Embedding Service", lifespan=lifespan)


@app.get("/health")
def health():
    return {"status": "ok", "dimension": embedder.dimension if embedder else 0}


@app.post("/embed", response_model=EmbedResponse)
def embed(req: EmbedRequest):
    vectors = embedder.embed(req.texts)
    return EmbedResponse(vectors=vectors)
