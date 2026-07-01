from sentence_transformers import SentenceTransformer
import numpy as np

MODEL_NAME = "sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2"


class Embedder:
    def __init__(self) -> None:
        self.model = SentenceTransformer(MODEL_NAME)
        self.dimension = self.model.get_sentence_embedding_dimension()

    def embed(self, texts: list[str]) -> list[list[float]]:
        embeddings: np.ndarray = self.model.encode(
            texts,
            normalize_embeddings=True,
            show_progress_bar=False,
        )
        return embeddings.tolist()
